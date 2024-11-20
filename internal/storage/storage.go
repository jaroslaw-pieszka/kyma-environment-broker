package storage

import (
	"time"

	"github.com/gocraft/dbr"
	"github.com/sirupsen/logrus"

	eventsapi "github.com/kyma-project/kyma-environment-broker/common/events"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/driver/memory"
	postgres "github.com/kyma-project/kyma-environment-broker/internal/storage/driver/postsql"
	eventstorage "github.com/kyma-project/kyma-environment-broker/internal/storage/driver/postsql/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
)

type BrokerStorage interface {
	Instances() Instances
	Operations() Operations
	Provisioning() Provisioning
	Deprovisioning() Deprovisioning
	Orchestrations() Orchestrations
	RuntimeStates() RuntimeStates
	SubaccountStates() SubaccountStates
	Events() Events
	InstancesArchived() InstancesArchived
	Bindings() Bindings
}

const (
	connectionRetries = 10
)

func NewFromConfig(cfg Config, evcfg events.Config, cipher postgres.Cipher, log logrus.FieldLogger) (BrokerStorage, *dbr.Connection, error) {
	log.Infof("Setting DB connection pool params: connectionMaxLifetime=%s "+
		"maxIdleConnections=%d maxOpenConnections=%d", cfg.ConnMaxLifetime, cfg.MaxIdleConns, cfg.MaxOpenConns)

	connection, err := postsql.InitializeDatabase(cfg.ConnectionURL(), connectionRetries, log)
	if err != nil {
		return nil, nil, err
	}

	connection.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	connection.SetMaxIdleConns(cfg.MaxIdleConns)
	connection.SetMaxOpenConns(cfg.MaxOpenConns)

	fact := postsql.NewFactory(connection)

	operation := postgres.NewOperation(fact, cipher)
	return storage{
		instance:          postgres.NewInstance(fact, operation, cipher),
		operation:         operation,
		orchestrations:    postgres.NewOrchestrations(fact),
		runtimeStates:     postgres.NewRuntimeStates(fact, cipher),
		events:            events.New(evcfg, eventstorage.New(fact, log)),
		subaccountStates:  postgres.NewSubaccountStates(fact),
		instancesArchived: postgres.NewInstanceArchived(fact),
		bindings:          postgres.NewBinding(fact, cipher),
	}, connection, nil
}

func NewMemoryStorage() BrokerStorage {
	op := memory.NewOperation()
	ss := memory.NewSubaccountStates()
	return storage{
		operation:         op,
		subaccountStates:  ss,
		instance:          memory.NewInstance(op, ss),
		orchestrations:    memory.NewOrchestrations(),
		runtimeStates:     memory.NewRuntimeStates(),
		events:            events.New(events.Config{}, NewInMemoryEvents()),
		instancesArchived: memory.NewInstanceArchivedInMemoryStorage(),
		bindings:          memory.NewBinding(),
	}
}

type inMemoryEvents struct {
	events []eventsapi.EventDTO
	log    *logrus.Logger
}

func NewInMemoryEvents() *inMemoryEvents {
	return &inMemoryEvents{
		events: make([]eventsapi.EventDTO, 0),
		log:    logrus.New(),
	}
}

func (_ inMemoryEvents) RunGarbageCollection(pollingPeriod, retention time.Duration) {
	return
}

func (e *inMemoryEvents) InsertEvent(eventLevel eventsapi.EventLevel, message, instanceID, operationID string) {
	e.events = append(e.events, eventsapi.EventDTO{Level: eventLevel, InstanceID: &instanceID, OperationID: &operationID, Message: message})
	e.log.Printf("EVENT [%v/%v] %v: %v\n", instanceID, operationID, eventLevel, message)
}

func (e *inMemoryEvents) ListEvents(filter eventsapi.EventFilter) ([]eventsapi.EventDTO, error) {
	var events []eventsapi.EventDTO
	for _, ev := range e.events {
		if !requiredContains(ev.InstanceID, filter.InstanceIDs) {
			continue
		}
		if !requiredContains(ev.OperationID, filter.OperationIDs) {
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

func requiredContains[T comparable](el *T, sl []T) bool {
	if len(sl) == 0 {
		return true
	}
	if el == nil {
		return false
	}
	for _, x := range sl {
		if *el == x {
			return true
		}
	}
	return false
}

type storage struct {
	instance          Instances
	operation         Operations
	orchestrations    Orchestrations
	runtimeStates     RuntimeStates
	events            Events
	subaccountStates  SubaccountStates
	instancesArchived InstancesArchived
	bindings          Bindings
}

func (s storage) Instances() Instances {
	return s.instance
}

func (s storage) Operations() Operations {
	return s.operation
}

func (s storage) Provisioning() Provisioning {
	return s.operation
}

func (s storage) Deprovisioning() Deprovisioning {
	return s.operation
}

func (s storage) Orchestrations() Orchestrations {
	return s.orchestrations
}

func (s storage) RuntimeStates() RuntimeStates {
	return s.runtimeStates
}

func (s storage) Events() Events {
	return s.events
}

func (s storage) SubaccountStates() SubaccountStates {
	return s.subaccountStates
}

func (s storage) InstancesArchived() InstancesArchived {
	return s.instancesArchived
}

func (s storage) Bindings() Bindings {
	return s.bindings
}
