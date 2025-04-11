package update

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"k8s.io/apimachinery/pkg/api/errors"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process/infrastructure_manager"
	"github.com/kyma-project/kyma-environment-broker/internal/process/provisioning"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UpdateRuntimeStep struct {
	operationManager           *process.OperationManager
	k8sClient                  client.Client
	delay                      time.Duration
	config                     infrastructure_manager.InfrastructureManagerConfig
	useSmallerMachineTypes     bool
	trialPlatformRegionMapping map[string]string
	useAdditionalOIDCSchema    bool
}

func NewUpdateRuntimeStep(os storage.Operations, k8sClient client.Client, delay time.Duration, infrastructureManagerConfig infrastructure_manager.InfrastructureManagerConfig, trialPlatformRegionMapping map[string]string, useAdditionalOIDCSchema bool) *UpdateRuntimeStep {
	step := &UpdateRuntimeStep{
		k8sClient:                  k8sClient,
		delay:                      delay,
		config:                     infrastructureManagerConfig,
		useSmallerMachineTypes:     infrastructureManagerConfig.UseSmallerMachineTypes,
		trialPlatformRegionMapping: trialPlatformRegionMapping,
		useAdditionalOIDCSchema:    useAdditionalOIDCSchema,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

func (s *UpdateRuntimeStep) Name() string {
	return "Update_Runtime_Resource"
}

func (s *UpdateRuntimeStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	// Check if the runtime exists

	var runtime = imv1.Runtime{}
	err := s.k8sClient.Get(context.Background(), client.ObjectKey{Name: operation.GetRuntimeResourceName(), Namespace: operation.GetRuntimeResourceNamespace()}, &runtime)
	if err != nil {
		if errors.IsNotFound(err) {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("Runtime Resource  %s not found", operation.GetRuntimeResourceName()), err, log)
		}
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to get Runtime Resource %s", operation.GetRuntimeResourceName()), err, 10*time.Second, 1*time.Minute, log)
	}

	// Update the runtime

	runtime.Spec.Shoot.Provider.Workers[0].Machine.Type = provisioning.DefaultIfParamNotSet(runtime.Spec.Shoot.Provider.Workers[0].Machine.Type, operation.UpdatingParameters.MachineType)
	runtime.Spec.Shoot.Provider.Workers[0].Minimum = int32(provisioning.DefaultIfParamNotSet(int(runtime.Spec.Shoot.Provider.Workers[0].Minimum), operation.UpdatingParameters.AutoScalerMin))
	runtime.Spec.Shoot.Provider.Workers[0].Maximum = int32(provisioning.DefaultIfParamNotSet(int(runtime.Spec.Shoot.Provider.Workers[0].Maximum), operation.UpdatingParameters.AutoScalerMax))

	maxSurge := intstr.FromInt32(int32(provisioning.DefaultIfParamNotSet(runtime.Spec.Shoot.Provider.Workers[0].MaxSurge.IntValue(), operation.UpdatingParameters.MaxSurge)))
	runtime.Spec.Shoot.Provider.Workers[0].MaxSurge = &maxSurge
	maxUnavailable := intstr.FromInt32(int32(provisioning.DefaultIfParamNotSet(runtime.Spec.Shoot.Provider.Workers[0].MaxUnavailable.IntValue(), operation.UpdatingParameters.MaxUnavailable)))
	runtime.Spec.Shoot.Provider.Workers[0].MaxUnavailable = &maxUnavailable

	if operation.UpdatingParameters.AdditionalWorkerNodePools != nil {
		values, err := provider.GetPlanSpecificValues(&operation, s.config.MultiZoneCluster, s.config.DefaultTrialProvider, s.config.UseSmallerMachineTypes, s.trialPlatformRegionMapping,
			s.config.DefaultGardenerShootPurpose, s.config.ControlPlaneFailureTolerance)
		if err != nil {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("while calculating plan specific values: %s", err), err, log)
		}

		currentAdditionalWorkers := make(map[string]gardener.Worker)
		if runtime.Spec.Shoot.Provider.AdditionalWorkers != nil {
			for _, worker := range *runtime.Spec.Shoot.Provider.AdditionalWorkers {
				currentAdditionalWorkers[worker.Name] = worker
			}
		}

		additionalWorkers := provisioning.CreateAdditionalWorkers(s.config, values, currentAdditionalWorkers, operation.UpdatingParameters.AdditionalWorkerNodePools, runtime.Spec.Shoot.Provider.Workers[0].Zones)
		runtime.Spec.Shoot.Provider.AdditionalWorkers = &additionalWorkers
	}

	if oidc := operation.UpdatingParameters.OIDC; oidc != nil {
		if oidc.List != nil {
			oidcConfigs := make([]gardener.OIDCConfig, 0)
			for _, oidcConfig := range oidc.List {
				requiredClaims := make(map[string]string)
				for _, claim := range oidcConfig.RequiredClaims {
					parts := strings.SplitN(claim, "=", 2)
					if len(parts) == 2 {
						requiredClaims[parts[0]] = parts[1]
					}
				}
				oidcConfigs = append(oidcConfigs, gardener.OIDCConfig{
					ClientID:       &oidcConfig.ClientID,
					IssuerURL:      &oidcConfig.IssuerURL,
					SigningAlgs:    oidcConfig.SigningAlgs,
					GroupsClaim:    &oidcConfig.GroupsClaim,
					UsernamePrefix: &oidcConfig.UsernamePrefix,
					UsernameClaim:  &oidcConfig.UsernameClaim,
					RequiredClaims: requiredClaims,
					GroupsPrefix:   &oidcConfig.GroupsPrefix,
				})
			}
			runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = &oidcConfigs
		} else if dto := oidc.OIDCConfigDTO; dto != nil {
			if runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig == nil {
				runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = &[]gardener.OIDCConfig{{}}
			}
			config := &(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0]
			if len(dto.SigningAlgs) > 0 {
				config.SigningAlgs = dto.SigningAlgs
			}
			if dto.ClientID != "" {
				config.ClientID = &dto.ClientID
			}
			if dto.IssuerURL != "" {
				config.IssuerURL = &dto.IssuerURL
			}
			if dto.GroupsClaim != "" {
				config.GroupsClaim = &dto.GroupsClaim
			}
			if dto.UsernamePrefix != "" {
				config.UsernamePrefix = &dto.UsernamePrefix
			}
			if dto.UsernameClaim != "" {
				config.UsernameClaim = &dto.UsernameClaim
			}
			if dto.GroupsPrefix != "" {
				config.GroupsPrefix = &dto.GroupsPrefix
			}
			if s.useAdditionalOIDCSchema && len(dto.RequiredClaims) > 0 {
				requiredClaims := make(map[string]string)
				for _, claim := range dto.RequiredClaims {
					parts := strings.SplitN(claim, "=", 2)
					if len(parts) == 2 {
						requiredClaims[parts[0]] = parts[1]
					}
				}
				config.RequiredClaims = requiredClaims
			}
		}
	}

	// operation.ProvisioningParameters were calculated and joined across provisioning and all update operations
	if len(operation.ProvisioningParameters.Parameters.RuntimeAdministrators) != 0 {
		// prepare new admins list for existing runtime
		newAdministrators := make([]string, 0, len(operation.ProvisioningParameters.Parameters.RuntimeAdministrators))
		newAdministrators = append(newAdministrators, operation.ProvisioningParameters.Parameters.RuntimeAdministrators...)

		runtime.Spec.Security.Administrators = newAdministrators
	} else {
		if operation.ProvisioningParameters.ErsContext.UserID != "" {
			// get default admin (user_id from provisioning operation)
			runtime.Spec.Security.Administrators = []string{operation.ProvisioningParameters.ErsContext.UserID}
		} else {
			// some old clusters does not have an user_id
			runtime.Spec.Security.Administrators = []string{}
		}
	}

	if operation.ProvisioningParameters.ErsContext.LicenseType != nil {
		disabled := *operation.ProvisioningParameters.ErsContext.DisableEnterprisePolicyFilter()
		runtime.Spec.Security.Networking.Filter.Egress.Enabled = !disabled
	}

	err = s.k8sClient.Update(context.Background(), &runtime)
	if err != nil {
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to update Runtime Resource %s", operation.GetRuntimeResourceName()), err, 10*time.Second, 1*time.Minute, log)
	}

	// this sleep is needed to wait for the runtime to be updated by the infrastructure manager with state PENDING,
	// then we can wait for the state READY in the next step
	time.Sleep(s.delay)

	return operation, 0, nil
}
