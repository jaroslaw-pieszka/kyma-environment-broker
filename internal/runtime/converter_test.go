package runtime

import (
	"reflect"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
)

func TestConverting_Provisioning(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.InProgress, time.Now()))

	// then
	assert.Equal(t, runtime.StateProvisioning, dto.Status.State)
}

func TestConverting_Provisioned(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))

	// then
	assert.Equal(t, runtime.StateSucceeded, dto.Status.State)
}

func TestConverting_ProvisioningFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Failed, time.Now()))

	// then
	assert.Equal(t, runtime.StateFailed, dto.Status.State)
}

func TestConverting_Updating(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt:     time.Now().Add(time.Second),
			ID:            "prov-id",
			State:         domain.InProgress,
			UpdatedPlanID: broker.BuildRuntimeAWSPlanID,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateUpdating, dto.Status.State)
	assert.Equal(t, broker.BuildRuntimeAWSPlanName, dto.Status.Update.Data[0].UpdatedPlanName)
}

func TestConverting_UpdateFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(time.Second),
			ID:        "prov-id",
			State:     domain.Failed,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateError, dto.Status.State)
}

func TestConverting_Suspending(t *testing.T) {
	t.Run("last operation should be deprovisioning", func(t *testing.T) {
		// given
		instance := fixInstance()
		svc := NewConverter("eu")

		// when
		dto, _ := svc.NewDTO(instance)
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
		svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.InProgress, time.Now().Add(time.Second)))

		// then
		assert.Equal(t, runtime.StateDeprovisioning, dto.Status.State)
	})

	t.Run("last operation should not be deprovisioning when it is pending", func(t *testing.T) {
		// given
		instance := fixInstance()
		svc := NewConverter("eu")

		// when
		dto, _ := svc.NewDTO(instance)
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.InProgress, time.Now()))
		svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(internal.OperationStatePending, time.Now().Add(time.Second)))

		// then
		assert.Equal(t, runtime.StateProvisioning, dto.Status.State)
	})
}

func TestConverting_Deprovisioning(t *testing.T) {
	t.Run("last operation should be deprovisioning", func(t *testing.T) {
		// given
		instance := fixInstance()
		svc := NewConverter("eu")

		// when
		dto, _ := svc.NewDTO(instance)
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
		svc.ApplyDeprovisioningOperation(&dto, fixDeprovisionOperation(domain.InProgress, time.Now().Add(time.Second)))

		// then
		assert.Equal(t, runtime.StateDeprovisioning, dto.Status.State)
	})

	t.Run("last operation should not be deprovisioning when it is pending", func(t *testing.T) {
		// given
		instance := fixInstance()
		svc := NewConverter("eu")

		// when
		dto, _ := svc.NewDTO(instance)
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.InProgress, time.Now()))
		svc.ApplyDeprovisioningOperation(&dto, fixDeprovisionOperation(internal.OperationStatePending, time.Now().Add(time.Second)))

		// then
		assert.Equal(t, runtime.StateProvisioning, dto.Status.State)
	})
}

func TestConverting_DeprovisionFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplyDeprovisioningOperation(&dto, fixDeprovisionOperation(domain.Failed, time.Now().Add(time.Second)))

	// then
	assert.Equal(t, runtime.StateFailed, dto.Status.State)
}

func TestConverting_SuspendFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.Failed, time.Now().Add(time.Second)))

	// then
	assert.Equal(t, runtime.StateFailed, dto.Status.State)
}

func TestConverting_SuspendedAndUpdated(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.Succeeded, time.Now().Add(time.Second)))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(2 * time.Second),
			ID:        "prov-id",
			State:     domain.Succeeded,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateSuspended, dto.Status.State)
}

func TestConverting_SuspendedAndUpdateFAiled(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.Succeeded, time.Now().Add(time.Second)))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(2 * time.Second),
			ID:        "prov-id",
			State:     domain.Failed,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateSuspended, dto.Status.State)
}

func TestConverting_ProvisioningOperationConverter(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)

	//expected stages in order
	expected := []string{"start", "create_runtime", "check_kyma", "post_actions"}

	t.Run("finished orders should be not set", func(t *testing.T) {
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))

		// then
		assert.Equal(t, []string(nil), dto.Status.Provisioning.FinishedStages)
	})

	t.Run("finished orders should be set in order", func(t *testing.T) {
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperationWithStagesAndVersion(domain.Succeeded, time.Now()))

		// then
		assert.True(t, reflect.DeepEqual(expected, dto.Status.Provisioning.FinishedStages))
	})
}

func TestConverting_ProvisioningParams(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)

	// then
	assert.Equal(t, instance.Parameters.Parameters, dto.Parameters)
}

func fixSuspensionOperation(state domain.LastOperationState, createdAt time.Time) []internal.DeprovisioningOperation {
	return []internal.DeprovisioningOperation{{
		Operation: internal.Operation{
			CreatedAt: createdAt,
			ID:        "s-id",
			State:     state,
			Temporary: true,
		},
	}}
}

func fixDeprovisionOperation(state domain.LastOperationState, createdAt time.Time) *internal.DeprovisioningOperation {
	return &internal.DeprovisioningOperation{
		Operation: internal.Operation{
			CreatedAt: createdAt,
			ID:        "s-id",
			State:     state,
		},
	}
}

func fixInstance() internal.Instance {
	return internal.Instance{
		InstanceID:                  "instance-id",
		RuntimeID:                   "runtime-id",
		GlobalAccountID:             "global-account-id",
		SubscriptionGlobalAccountID: "subgid",
		SubAccountID:                "sub-account-id",
		Parameters: internal.ProvisioningParameters{
			Parameters: runtime.ProvisioningParametersDTO{
				Name: "instance-name",
			},
		},
	}
}

func fixProvisioningOperation(state domain.LastOperationState, createdAt time.Time) *internal.ProvisioningOperation {
	return &internal.ProvisioningOperation{
		Operation: internal.Operation{
			CreatedAt: createdAt,
			ID:        "prov-id",
			State:     state,
		},
	}
}

func fixProvisioningOperationWithStagesAndVersion(state domain.LastOperationState, createdAt time.Time) *internal.ProvisioningOperation {
	return &internal.ProvisioningOperation{
		Operation: internal.Operation{
			CreatedAt:      createdAt,
			ID:             "prov-id",
			State:          state,
			FinishedStages: []string{"start", "create_runtime", "check_kyma", "post_actions"},
		},
	}
}
