package orchestration

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	brokerapi "github.com/pivotal-cf/brokerapi/v12/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// RuntimeLister is the interface to get runtime objects from KEB
//
//go:generate mockery --name=RuntimeLister --output=. --outpkg=orchestration --case=underscore --structname RuntimeListerMock --filename runtime_lister_mock.go
type RuntimeLister interface {
	ListAllRuntimes() ([]runtime.RuntimeDTO, error)
}

// GardenerRuntimeResolver is the default resolver which implements the RuntimeResolver interface.
// This resolver uses the Shoot resources on the Gardener cluster to resolve the runtime targets.
//
// Naive implementation, listing all the shoots and perfom filtering on the result.
// The logic could be optimized with k8s client cache using shoot lister / indexer.
// The implementation is thread safe, i.e. it is safe to call Resolve() from multiple threads concurrently.
type GardenerRuntimeResolver struct {
	gardenerClient    dynamic.Interface
	gardenerNamespace string
	runtimeLister     RuntimeLister
	runtimes          map[string]runtime.RuntimeDTO
	mutex             sync.RWMutex
	logger            *slog.Logger
}

const (
	globalAccountLabel             = "account"
	subAccountLabel                = "subaccount"
	runtimeIDAnnotationProvisioner = "kcp.provisioner.kyma-project.io/runtime-id"
	runtimeIDAnnotationKim         = "infrastructuremanager.kyma-project.io/runtime-id"
	maintenanceWindowFormat        = "150405-0700"
)

// NewGardenerRuntimeResolver constructs a GardenerRuntimeResolver with the mandatory input parameters.
func NewGardenerRuntimeResolver(gardenerClient dynamic.Interface, gardenerNamespace string, lister RuntimeLister, logger *slog.Logger) *GardenerRuntimeResolver {
	return &GardenerRuntimeResolver{
		gardenerClient:    gardenerClient,
		gardenerNamespace: gardenerNamespace,
		runtimeLister:     lister,
		runtimes:          map[string]runtime.RuntimeDTO{},
		logger:            logger.With("orchestration", "resolver"),
	}
}

// Resolve given an input slice of target specs to include and exclude, returns back a list of unique Runtime objects
func (resolver *GardenerRuntimeResolver) Resolve(targets TargetSpec) ([]Runtime, error) {
	runtimeIncluded := map[string]bool{}
	runtimeExcluded := map[string]bool{}
	runtimes := []Runtime{}
	shoots, err := resolver.getAllShoots()
	if err != nil {
		return nil, fmt.Errorf("while listing gardener shoots in namespace %s: %w", resolver.gardenerNamespace, err)
	}
	err = resolver.syncRuntimeOperations()
	if err != nil {
		return nil, fmt.Errorf("while syncing runtimes: %w", err)
	}

	// Assemble IDs of runtimes to exclude
	for _, rt := range targets.Exclude {
		runtimesToExclude, err := resolver.resolveRuntimeTarget(rt, shoots)
		if err != nil {
			return nil, err
		}
		for _, r := range runtimesToExclude {
			runtimeExcluded[r.RuntimeID] = true
		}
	}

	// Include runtimes which are not excluded
	for _, rt := range targets.Include {
		runtimesToAdd, err := resolver.resolveRuntimeTarget(rt, shoots)
		if err != nil {
			return nil, err
		}
		for _, r := range runtimesToAdd {
			if !runtimeExcluded[r.RuntimeID] && !runtimeIncluded[r.RuntimeID] {
				runtimeIncluded[r.RuntimeID] = true
				runtimes = append(runtimes, r)
			}
		}
	}

	return runtimes, nil
}

func (resolver *GardenerRuntimeResolver) getAllShoots() ([]unstructured.Unstructured, error) {
	ctx := context.Background()
	shootList, err := resolver.gardenerClient.Resource(gardener.ShootResource).Namespace(resolver.gardenerNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return shootList.Items, nil
}

func (resolver *GardenerRuntimeResolver) syncRuntimeOperations() error {
	runtimes, err := resolver.runtimeLister.ListAllRuntimes()
	if err != nil {
		return err
	}
	resolver.mutex.Lock()
	defer resolver.mutex.Unlock()

	for _, rt := range runtimes {
		resolver.runtimes[rt.RuntimeID] = rt
	}

	return nil
}

func (resolver *GardenerRuntimeResolver) getRuntime(runtimeID string) (runtime.RuntimeDTO, bool) {
	resolver.mutex.RLock()
	defer resolver.mutex.RUnlock()
	rt, ok := resolver.runtimes[runtimeID]

	return rt, ok
}

func (resolver *GardenerRuntimeResolver) resolveRuntimeTarget(rt RuntimeTarget, shoots []unstructured.Unstructured) ([]Runtime, error) {
	runtimes := []Runtime{}
	// Iterate over all shoots. Evaluate target specs. If multiple are specified, all must match for a given shoot.
	for _, s := range shoots {
		shoot := &gardener.Shoot{Unstructured: s}
		runtimeID := shoot.GetAnnotations()[runtimeIDAnnotationProvisioner]
		if runtimeID == "" {
			runtimeID = shoot.GetAnnotations()[runtimeIDAnnotationKim]
			if runtimeID == "" {
				resolver.logger.Error(fmt.Sprintf("Failed to get runtimeID neither from %s annotation nor from %s for Shoot %s", runtimeIDAnnotationProvisioner, runtimeIDAnnotationKim, shoot.GetName()))
				continue
			}
		}
		r, ok := resolver.getRuntime(runtimeID)
		if !ok {
			resolver.logger.Error(fmt.Sprintf("Couldn't find runtime for runtimeID %s", runtimeID))
			continue
		}

		lastOp := r.LastOperation()
		// Skip runtimes for which the last operation is
		//  - not succeeded provision or unsuspension
		//  - suspension
		//  - deprovision
		if lastOp.Type == runtime.Deprovision || lastOp.Type == runtime.Suspension || (lastOp.Type == runtime.Provision || lastOp.Type == runtime.Unsuspension) && lastOp.State != string(brokerapi.Succeeded) {
			resolver.logger.Info(fmt.Sprintf("Skipping Shoot %s (runtimeID: %s, instanceID %s) due to %s state: %s", shoot.GetName(), runtimeID, r.InstanceID, lastOp.Type, lastOp.State))
			continue
		}
		maintenanceWindowBegin, err := time.Parse(maintenanceWindowFormat, shoot.GetSpecMaintenanceTimeWindowBegin())
		if err != nil {
			resolver.logger.Error(fmt.Sprintf("Failed to parse maintenanceWindowBegin value %s of shoot %s ", shoot.GetSpecMaintenanceTimeWindowBegin(), shoot.GetName()))
			continue
		}
		maintenanceWindowEnd, err := time.Parse(maintenanceWindowFormat, shoot.GetSpecMaintenanceTimeWindowEnd())
		if err != nil {
			resolver.logger.Error(fmt.Sprintf("Failed to parse maintenanceWindowEnd value %s of shoot %s ", shoot.GetSpecMaintenanceTimeWindowEnd(), shoot.GetName()))
			continue
		}

		// Match exact shoot by runtimeID
		if rt.RuntimeID != "" {
			if rt.RuntimeID == runtimeID {
				runtimes = append(runtimes, resolver.runtimeFromDTO(r, shoot.GetName(), maintenanceWindowBegin, maintenanceWindowEnd))
			}
			continue
		}

		// Match exact shoot by instanceID
		if rt.InstanceID != "" {
			if rt.InstanceID != r.InstanceID {
				continue
			}
		}

		// Match exact shoot by name
		if rt.Shoot != "" && rt.Shoot != shoot.GetName() {
			continue
		}

		// Perform match against a specific PlanName
		if rt.PlanName != "" {
			if rt.PlanName != r.ServicePlanName {
				continue
			}
		}

		// Perform match against GlobalAccount regexp
		if rt.GlobalAccount != "" {
			matched, err := regexp.MatchString(rt.GlobalAccount, shoot.GetLabels()[globalAccountLabel])
			if err != nil || !matched {
				continue
			}
		}

		// Perform match against SubAccount regexp
		if rt.SubAccount != "" {
			matched, err := regexp.MatchString(rt.SubAccount, shoot.GetLabels()[subAccountLabel])
			if err != nil || !matched {
				continue
			}
		}

		// Perform match against Region regexp
		if rt.Region != "" {
			matched, err := regexp.MatchString(rt.Region, shoot.GetSpecRegion())
			if err != nil || !matched {
				continue
			}
		}

		// Check if target: all is specified
		if rt.Target != "" && rt.Target != TargetAll {
			continue
		}

		runtimes = append(runtimes, resolver.runtimeFromDTO(r, shoot.GetName(), maintenanceWindowBegin, maintenanceWindowEnd))
	}

	return runtimes, nil
}

func (*GardenerRuntimeResolver) runtimeFromDTO(runtime runtime.RuntimeDTO, shootName string, windowBegin, windowEnd time.Time) Runtime {
	return Runtime{
		InstanceID:             runtime.InstanceID,
		RuntimeID:              runtime.RuntimeID,
		GlobalAccountID:        runtime.GlobalAccountID,
		SubAccountID:           runtime.SubAccountID,
		Plan:                   runtime.ServicePlanName,
		Region:                 runtime.ProviderRegion,
		ShootName:              shootName,
		MaintenanceWindowBegin: windowBegin,
		MaintenanceWindowEnd:   windowEnd,
		MaintenanceDays:        []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
	}
}
