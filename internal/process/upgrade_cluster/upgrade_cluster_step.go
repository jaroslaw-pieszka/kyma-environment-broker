package upgrade_cluster

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

const DryRunPrefix = "dry_run-"

type UpgradeClusterStep struct {
	operationManager    *process.UpgradeClusterOperationManager
	provisionerClient   provisioner.Client
	runtimeStateStorage storage.RuntimeStates
	timeSchedule        TimeSchedule
}

func NewUpgradeClusterStep(
	os storage.Operations,
	runtimeStorage storage.RuntimeStates,
	cli provisioner.Client,
	timeSchedule *TimeSchedule) *UpgradeClusterStep {
	ts := timeSchedule
	if ts == nil {
		ts = &TimeSchedule{
			Retry:                 5 * time.Second,
			StatusCheck:           time.Minute,
			UpgradeClusterTimeout: time.Hour,
		}
	}

	return &UpgradeClusterStep{
		operationManager:    process.NewUpgradeClusterOperationManager(os),
		provisionerClient:   cli,
		runtimeStateStorage: runtimeStorage,
		timeSchedule:        *ts,
	}
}

func (s *UpgradeClusterStep) Name() string {
	return "Upgrade_Cluster"
}

func (s *UpgradeClusterStep) Run(operation internal.UpgradeClusterOperation, log *slog.Logger) (internal.UpgradeClusterOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > s.timeSchedule.UpgradeClusterTimeout {
		log.Info(fmt.Sprintf("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt))
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.timeSchedule.UpgradeClusterTimeout), nil, log)
	}

	latestRuntimeStateWithOIDC, err := s.runtimeStateStorage.GetLatestWithOIDCConfigByRuntimeID(operation.InstanceDetails.RuntimeID)
	if err != nil {
		return s.operationManager.RetryOperation(operation, err.Error(), err, 5*time.Second, 1*time.Minute, log)
	}

	input, err := s.createUpgradeShootInput(operation, &latestRuntimeStateWithOIDC.ClusterConfig)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create upgradeShoot input", err, log)
	}

	if operation.DryRun {
		// runtimeID is set with prefix to indicate the fake runtime state
		err = s.runtimeStateStorage.Insert(
			internal.NewRuntimeState(fmt.Sprintf("%s%s", DryRunPrefix, operation.RuntimeOperation.RuntimeID), operation.Operation.ID, nil, gardenerUpgradeInputToConfigInput(input)),
		)
		if err != nil {
			return operation, 10 * time.Second, nil
		}
		return s.operationManager.OperationSucceeded(operation, "dry run succeeded", log)
	}

	var provisionerResponse gqlschema.OperationStatus
	if operation.ProvisionerOperationID == "" {
		// trigger upgradeRuntime mutation
		provisionerResponse, err = s.provisionerClient.UpgradeShoot(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeOperation.RuntimeID, input)
		if err != nil {
			log.Error(fmt.Sprintf("call to provisioner failed: %s", err))
			return operation, s.timeSchedule.Retry, nil
		}

		repeat := time.Duration(0)
		operation, repeat, _ = s.operationManager.UpdateOperation(operation, func(op *internal.UpgradeClusterOperation) {
			op.ProvisionerOperationID = *provisionerResponse.ID
			op.Description = "cluster upgrade in progress"
		}, log)
		if repeat != 0 {
			log.Error("cannot save operation ID from provisioner")
			return operation, s.timeSchedule.Retry, nil
		}
	}

	if provisionerResponse.RuntimeID == nil {
		provisionerResponse, err = s.provisionerClient.RuntimeOperationStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.ProvisionerOperationID)
		if err != nil {
			log.Error(fmt.Sprintf("call to provisioner about operation status failed: %s", err))
			return operation, s.timeSchedule.Retry, nil
		}
	}
	if provisionerResponse.RuntimeID == nil {
		return operation, s.timeSchedule.StatusCheck, nil
	}
	log = log.With("runtimeID", *provisionerResponse.RuntimeID)
	log.Info(fmt.Sprintf("call to provisioner for upgrade succeeded, got operation ID %q", *provisionerResponse.ID))

	rs := internal.NewRuntimeState(*provisionerResponse.RuntimeID, operation.Operation.ID, nil, gardenerUpgradeInputToConfigInput(input))
	err = s.runtimeStateStorage.Insert(rs)
	if err != nil {
		log.Error(fmt.Sprintf("cannot insert runtimeState: %s", err))
		return operation, 10 * time.Second, nil
	}

	log.Info("cluster upgrade process initiated successfully")

	// return repeat mode to start the initialization step which will now check the runtime status
	return operation, s.timeSchedule.Retry, nil

}

func (s *UpgradeClusterStep) createUpgradeShootInput(operation internal.UpgradeClusterOperation, lastClusterConfig *gqlschema.GardenerConfigInput) (gqlschema.UpgradeShootInput, error) {
	operation.InputCreator.SetProvisioningParameters(operation.ProvisioningParameters)
	if lastClusterConfig.OidcConfig != nil {
		operation.InputCreator.SetOIDCLastValues(*lastClusterConfig.OidcConfig)
	}
	input, err := operation.InputCreator.CreateUpgradeShootInput()
	if err != nil {
		return input, fmt.Errorf("while building upgradeShootInput for provisioner: %w", err)
	}
	input.GardenerConfig.ShootNetworkingFilterDisabled = operation.ProvisioningParameters.ErsContext.DisableEnterprisePolicyFilter()

	return input, nil
}

func gardenerUpgradeInputToConfigInput(input gqlschema.UpgradeShootInput) *gqlschema.GardenerConfigInput {
	disabled := false
	result := &gqlschema.GardenerConfigInput{
		MachineImage:                        input.GardenerConfig.MachineImage,
		MachineImageVersion:                 input.GardenerConfig.MachineImageVersion,
		DiskType:                            input.GardenerConfig.DiskType,
		VolumeSizeGb:                        input.GardenerConfig.VolumeSizeGb,
		Purpose:                             input.GardenerConfig.Purpose,
		OidcConfig:                          input.GardenerConfig.OidcConfig,
		EnableKubernetesVersionAutoUpdate:   input.GardenerConfig.EnableKubernetesVersionAutoUpdate,
		EnableMachineImageVersionAutoUpdate: input.GardenerConfig.EnableMachineImageVersionAutoUpdate,
		ShootNetworkingFilterDisabled:       &disabled,
	}
	if input.GardenerConfig.KubernetesVersion != nil {
		result.KubernetesVersion = *input.GardenerConfig.KubernetesVersion
	}
	if input.GardenerConfig.MachineType != nil {
		result.MachineType = *input.GardenerConfig.MachineType
	}
	if input.GardenerConfig.AutoScalerMin != nil {
		result.AutoScalerMin = *input.GardenerConfig.AutoScalerMin
	}
	if input.GardenerConfig.AutoScalerMax != nil {
		result.AutoScalerMax = *input.GardenerConfig.AutoScalerMax
	}
	if input.GardenerConfig.MaxSurge != nil {
		result.MaxSurge = *input.GardenerConfig.MaxSurge
	}
	if input.GardenerConfig.MaxUnavailable != nil {
		result.MaxUnavailable = *input.GardenerConfig.MaxUnavailable
	}
	result.ShootNetworkingFilterDisabled = input.GardenerConfig.ShootNetworkingFilterDisabled

	return result
}
