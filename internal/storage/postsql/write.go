package postsql

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/events"
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"

	"github.com/gocraft/dbr"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	UniqueViolationErrorCode = "23505"
)

type writeSession struct {
	session     *dbr.Session
	transaction *dbr.Tx
}

func (ws writeSession) UpdateInstanceLastOperation(instanceID, operationID string) error {
	_, err := ws.update(InstancesTableName).
		Set("last_operation_id", operationID).
		Where(dbr.Eq("instance_id", instanceID)).
		Exec()

	if err != nil {
		return dberr.Internal("Failed to update instance with last operation id: %s", err)
	}
	return nil
}

func (ws writeSession) DeleteBinding(instanceID, bindingID string) dberr.Error {
	_, err := ws.deleteFrom(BindingsTableName).
		Where(dbr.Eq("id", bindingID)).
		Where(dbr.Eq("instance_id", instanceID)).
		Exec()

	if err != nil {
		return dberr.Internal("Failed to delete record from bindings table: %s", err)
	}
	return nil
}

func (ws writeSession) InsertBinding(binding dbmodel.BindingDTO) dberr.Error {
	_, err := ws.insertInto(BindingsTableName).
		Pair("id", binding.ID).
		Pair("instance_id", binding.InstanceID).
		Pair("created_at", binding.CreatedAt).
		Pair("expires_at", binding.ExpiresAt).
		Pair("kubeconfig", binding.Kubeconfig).
		Pair("expiration_seconds", binding.ExpirationSeconds).
		Pair("created_by", binding.CreatedBy).
		Exec()

	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == UniqueViolationErrorCode {
				return dberr.AlreadyExists("binding with id %s already exist for runtime %s", binding.ID, binding.InstanceID)
			}
		}
		return dberr.Internal("Failed to insert record to Binding table: %s", err)
	}

	return nil
}

func (ws writeSession) UpdateBinding(binding dbmodel.BindingDTO) dberr.Error {
	_, err := ws.update(BindingsTableName).
		Set("kubeconfig", binding.Kubeconfig).
		Set("expires_at", binding.ExpiresAt).
		Where(dbr.Eq("id", binding.ID)).
		Where(dbr.Eq("instance_id", binding.InstanceID)).
		Exec()

	if err != nil {
		return dberr.Internal("Failed to update record to Binding table: %s", err)
	}

	return nil
}

func (ws writeSession) InsertInstanceArchived(instance dbmodel.InstanceArchivedDTO) dberr.Error {
	_, err := ws.insertInto(InstancesArchivedTableName).
		Pair("instance_id", instance.InstanceID).
		Pair("last_runtime_id", instance.LastRuntimeID).
		Pair("global_account_id", instance.GlobalAccountID).
		Pair("subscription_global_account_id", instance.SubscriptionGlobalAccountID).
		Pair("subaccount_id", instance.SubaccountID).
		Pair("plan_id", instance.PlanID).
		Pair("plan_name", instance.PlanName).
		Pair("internal_user", instance.InternalUser).
		Pair("region", instance.Region).
		Pair("subaccount_region", instance.SubaccountRegion).
		Pair("provider", instance.Provider).
		Pair("shoot_name", instance.ShootName).
		Pair("provisioning_started_at", instance.ProvisioningStartedAt).
		Pair("provisioning_finished_at", instance.ProvisioningFinishedAt).
		Pair("provisioning_state", instance.ProvisioningState).
		Pair("first_deprovisioning_started_at", instance.FirstDeprovisioningStartedAt).
		Pair("first_deprovisioning_finished_at", instance.FirstDeprovisioningFinishedAt).
		Pair("last_deprovisioning_finished_at", instance.LastDeprovisioningFinishedAt).
		Exec()

	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == UniqueViolationErrorCode {
				return dberr.AlreadyExists("instance archived with id %s already exist", instance.InstanceID)
			}
		}
		return dberr.Internal("Failed to insert record to Instance table: %s", err)
	}

	return nil
}

func (ws writeSession) InsertInstance(instance dbmodel.InstanceDTO) dberr.Error {
	_, err := ws.insertInto(InstancesTableName).
		Pair("instance_id", instance.InstanceID).
		Pair("runtime_id", instance.RuntimeID).
		Pair("global_account_id", instance.GlobalAccountID).
		Pair("subscription_global_account_id", instance.SubscriptionGlobalAccountID).
		Pair("sub_account_id", instance.SubAccountID).
		Pair("service_id", instance.ServiceID).
		Pair("service_name", instance.ServiceName).
		Pair("service_plan_id", instance.ServicePlanID).
		Pair("service_plan_name", instance.ServicePlanName).
		Pair("dashboard_url", instance.DashboardURL).
		Pair("provisioning_parameters", instance.ProvisioningParameters).
		Pair("provider_region", instance.ProviderRegion).
		Pair("provider", instance.Provider).
		Pair("deleted_at", instance.DeletedAt).
		Pair("expired_at", instance.ExpiredAt).
		Pair("version", instance.Version).
		Exec()

	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == UniqueViolationErrorCode {
				return dberr.AlreadyExists("operation with id %s already exist", instance.InstanceID)
			}
		}
		return dberr.Internal("Failed to insert record to Instance table: %s", err)
	}

	return nil
}

func (ws writeSession) DeleteInstance(instanceID string) dberr.Error {
	_, err := ws.deleteFrom(InstancesTableName).
		Where(dbr.Eq("instance_id", instanceID)).
		Exec()

	if err != nil {
		return dberr.Internal("Failed to delete record from Instance table: %s", err)
	}
	return nil
}

func (ws writeSession) UpdateInstance(instance dbmodel.InstanceDTO) dberr.Error {
	res, err := ws.update(InstancesTableName).
		Where(dbr.Eq("instance_id", instance.InstanceID)).
		Where(dbr.Eq("version", instance.Version)).
		Set("instance_id", instance.InstanceID).
		Set("runtime_id", instance.RuntimeID).
		Set("global_account_id", instance.GlobalAccountID).
		Set("subscription_global_account_id", instance.SubscriptionGlobalAccountID).
		Set("service_id", instance.ServiceID).
		Set("service_plan_id", instance.ServicePlanID).
		Set("service_plan_name", instance.ServicePlanName).
		Set("dashboard_url", instance.DashboardURL).
		Set("provisioning_parameters", instance.ProvisioningParameters).
		Set("provider_region", instance.ProviderRegion).
		Set("provider", instance.Provider).
		Set("updated_at", time.Now()).
		Set("deleted_at", instance.DeletedAt).
		Set("version", instance.Version+1).
		Set("expired_at", instance.ExpiredAt).
		Exec()
	if err != nil {
		return dberr.Internal("Failed to update record to Instance table: %s", err)
	}
	rAffected, err := res.RowsAffected()
	if err != nil {
		// the optimistic locking requires numbers of rows affected
		return dberr.Internal("the DB driver does not support RowsAffected operation")
	}
	if rAffected == int64(0) {
		return dberr.NotFound("Cannot find Instance with ID:'%s' Version: %v", instance.InstanceID, instance.Version)
	}

	return nil
}

func (ws writeSession) InsertOperation(op dbmodel.OperationDTO) dberr.Error {
	_, err := ws.insertInto(OperationTableName).
		Pair("id", op.ID).
		Pair("instance_id", op.InstanceID).
		Pair("version", op.Version).
		Pair("created_at", op.CreatedAt).
		Pair("updated_at", op.UpdatedAt).
		Pair("description", op.Description).
		Pair("state", op.State).
		Pair("target_operation_id", op.TargetOperationID).
		Pair("type", op.Type).
		Pair("data", op.Data).
		Pair("provisioning_parameters", op.ProvisioningParameters.String).
		Pair("finished_stages", op.FinishedStages).
		Exec()

	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == UniqueViolationErrorCode {
				return dberr.AlreadyExists("operation with id %s already exist", op.ID)
			}
		}
		return dberr.Internal("Failed to insert record to operations table: %s", err)
	}

	return nil
}

func (ws writeSession) UpsertSubaccountState(state dbmodel.SubaccountStateDTO) dberr.Error {
	result, err := ws.update(SubaccountStatesTableName).
		Where(dbr.Eq("id", state.ID)).
		Set("beta_enabled", state.BetaEnabled).
		Set("used_for_production", state.UsedForProduction).
		Set("modified_at", state.ModifiedAt).
		Exec()
	if err != nil {
		return dberr.Internal("Failed to update record to subaccount_states table: %s", err)
	}
	rAffected, err := result.RowsAffected()
	if rAffected == int64(0) {
		_, err = ws.insertInto(SubaccountStatesTableName).
			Pair("id", state.ID).
			Pair("beta_enabled", state.BetaEnabled).
			Pair("used_for_production", state.UsedForProduction).
			Pair("modified_at", state.ModifiedAt).
			Exec()
		if err != nil {
			return dberr.Internal("Failed to upsert record to subaccount_states table: %s", err)
		}
	}
	return nil
}

func (ws writeSession) DeleteState(subaccountID string) dberr.Error {
	_, err := ws.deleteFrom(SubaccountStatesTableName).
		Where(dbr.Eq("id", subaccountID)).
		Exec()
	if err != nil {
		return dberr.Internal("failed to delete state for subaccount %s: %v", subaccountID, err)
	}
	return nil
}

func (ws writeSession) UpdateOperation(op dbmodel.OperationDTO) dberr.Error {
	res, err := ws.update(OperationTableName).
		Where(dbr.Eq("id", op.ID)).
		Where(dbr.Eq("version", op.Version)).
		Set("instance_id", op.InstanceID).
		Set("version", op.Version+1).
		Set("created_at", op.CreatedAt).
		Set("updated_at", op.UpdatedAt).
		Set("description", op.Description).
		Set("state", op.State).
		Set("target_operation_id", op.TargetOperationID).
		Set("type", op.Type).
		Set("data", op.Data).
		Set("provisioning_parameters", op.ProvisioningParameters.String).
		Set("finished_stages", op.FinishedStages).
		Exec()

	if err != nil {
		if err == dbr.ErrNotFound {
			return dberr.NotFound("Cannot find Operation with ID:'%s'", op.ID)
		}
		return dberr.Internal("Failed to update record to Operation table: %s", err)
	}
	rAffected, e := res.RowsAffected()
	if e != nil {
		// the optimistic locking requires numbers of rows affected
		return dberr.Internal("the DB driver does not support RowsAffected operation")
	}
	if rAffected == int64(0) {
		return dberr.NotFound("Cannot find Operation with ID:'%s' Version: %v", op.ID, op.Version)
	}

	return nil
}

func (ws writeSession) InsertEvent(level events.EventLevel, message, instanceID, operationID string) dberr.Error {
	_, err := ws.insertInto("events").
		Pair("id", uuid.NewString()).
		Pair("level", level).
		Pair("instance_id", instanceID).
		Pair("operation_id", operationID).
		Pair("message", message).
		Pair("created_at", time.Now()).
		Exec()
	if err != nil {
		return dberr.Internal("Failed to insert event: %s", err)
	}
	return nil
}

func (ws writeSession) DeleteEvents(until time.Time) dberr.Error {
	_, err := ws.deleteFrom("events").
		Where(dbr.Lte("created_at", until)).
		Exec()
	if err != nil {
		return dberr.Internal("failed to delete events created until %v: %v", until.Format(time.RFC1123Z), err)
	}
	return nil
}

func (ws writeSession) DeleteOperationByID(id string) dberr.Error {
	_, err := ws.deleteFrom("operations").
		Where(dbr.Eq("id", id)).
		Exec()
	if err != nil {
		return dberr.Internal("unable to delete operation %s: %s", id, err.Error())
	}
	return nil
}

func (ws writeSession) InsertAction(actionType runtime.ActionType, instanceID, message, oldValue, newValue string) dberr.Error {
	_, err := ws.insertInto(ActionsTableName).
		Pair("id", uuid.NewString()).
		Pair("type", actionType).
		Pair("instance_id", instanceID).
		Pair("message", message).
		Pair("old_value", oldValue).
		Pair("new_value", newValue).
		Pair("created_at", time.Now()).
		Exec()
	if err != nil {
		return dberr.Internal("failed to insert action: %s", err)
	}
	return nil
}

func (ws writeSession) Commit() dberr.Error {
	err := ws.transaction.Commit()
	if err != nil {
		return dberr.Internal("Failed to commit transaction: %s", err)
	}

	return nil
}

func (ws writeSession) RollbackUnlessCommitted() {
	ws.transaction.RollbackUnlessCommitted()
}

func (ws writeSession) insertInto(table string) *dbr.InsertStmt {
	if ws.transaction != nil {
		return ws.transaction.InsertInto(table)
	}

	return ws.session.InsertInto(table)
}

func (ws writeSession) deleteFrom(table string) *dbr.DeleteStmt {
	if ws.transaction != nil {
		return ws.transaction.DeleteFrom(table)
	}

	return ws.session.DeleteFrom(table)
}

func (ws writeSession) update(table string) *dbr.UpdateStmt {
	if ws.transaction != nil {
		return ws.transaction.Update(table)
	}

	return ws.session.Update(table)
}
