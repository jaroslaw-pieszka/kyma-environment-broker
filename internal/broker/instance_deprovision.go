package broker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
)

type DeprovisionEndpoint struct {
	log *slog.Logger

	instancesStorage  storage.Instances
	operationsStorage storage.Deprovisioning

	queue Queue
}

func NewDeprovision(instancesStorage storage.Instances, operationsStorage storage.Operations, q Queue, log *slog.Logger) *DeprovisionEndpoint {
	return &DeprovisionEndpoint{
		log:               log.With("service", "DeprovisionEndpoint"),
		instancesStorage:  instancesStorage,
		operationsStorage: operationsStorage,

		queue: q,
	}
}

// Deprovision deletes an existing service instance
//
//	DELETE /v2/service_instances/{instance_id}
func (b *DeprovisionEndpoint) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, asyncAllowed bool) (domain.DeprovisionServiceSpec, error) {
	logger := b.log.With("instanceID", instanceID)
	logger.Info(fmt.Sprintf("Deprovisioning triggered, details: %+v", details))

	instance, err := b.instancesStorage.GetByID(instanceID)
	switch {
	case err == nil:
	case dberr.IsNotFound(err):
		logger.Warn("instance does not exist")
		return domain.DeprovisionServiceSpec{
			IsAsync: false,
		}, nil
	default:
		logger.Error(fmt.Sprintf("unable to get instance from storage: %s", err))
		return domain.DeprovisionServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("unable to get instance from the storage"), http.StatusInternalServerError, fmt.Sprintf("could not deprovision runtime, instanceID %s", instanceID))
	}

	logger = logger.With("runtimeID", instance.RuntimeID, "globalAccountID", instance.GlobalAccountID, "planID", instance.ServicePlanID)

	// check if operation with the same instance ID is already created
	existingOperation, errStorage := b.operationsStorage.GetDeprovisioningOperationByInstanceID(instanceID)
	if errStorage != nil && !dberr.IsNotFound(errStorage) {
		logger.Error(fmt.Sprintf("cannot get existing operation from storage %s", errStorage))
		return domain.DeprovisionServiceSpec{}, fmt.Errorf("cannot get existing operation from storage")
	}

	// temporary deprovisioning means suspension
	previousOperationIsNotTemporary := existingOperation != nil && !existingOperation.Temporary
	if previousOperationIsNotTemporary {
		switch {
		// there is an ongoing operation
		case existingOperation.State != domain.Failed && existingOperation.State != domain.Succeeded:
			logger.Info("deprovision operation already ongoing - not creating a new operation")
			return domain.DeprovisionServiceSpec{
				IsAsync:       true,
				OperationData: existingOperation.ID,
			}, nil
		case existingOperation.State == domain.Succeeded && len(existingOperation.ExcutedButNotCompleted) == 0:
			logger.Info("no steps to retry - not creating a new operation")
			return domain.DeprovisionServiceSpec{
				IsAsync:       true,
				OperationData: existingOperation.ID,
			}, nil
		}
	}

	// create and save new operation
	operationID := uuid.New().String()
	logger = logger.With("operationID", operationID)
	operation, err := internal.NewDeprovisioningOperationWithID(operationID, instance)
	if err != nil {
		logger.Error(fmt.Sprintf("cannot create new operation: %s", err))
		return domain.DeprovisionServiceSpec{}, fmt.Errorf("cannot create new operation")
	}
	if v := ctx.Value("User-Agent"); v != nil {
		operation.UserAgent = v.(string)
	}
	err = b.operationsStorage.InsertDeprovisioningOperation(operation)
	if err != nil {
		logger.Error(fmt.Sprintf("cannot save operation: %s", err))
		return domain.DeprovisionServiceSpec{}, fmt.Errorf("cannot save operation")
	}

	logger.Info("Adding operation to deprovisioning queue")
	b.queue.Add(operationID)

	return domain.DeprovisionServiceSpec{
		IsAsync:       true,
		OperationData: operationID,
	}, nil
}
