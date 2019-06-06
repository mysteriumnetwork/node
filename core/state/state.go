package state

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/state/event"
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/nat"
	natEvent "github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/session"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
)

type natStatusProvider interface {
	Status() nat.Status
	ConsumeNATEvent(event natEvent.Event)
}

type publisher interface {
	Publish(topic string, data interface{})
}

type serviceLister interface {
	List() map[service.ID]*service.Instance
}

type serviceSessionStorage interface {
	GetAll() []session.Session
}

type Keeper struct {
	state                 *stateEvent.State
	lock                  sync.RWMutex
	natStatusProvider     natStatusProvider
	publisher             publisher
	serviceLister         serviceLister
	serviceSessionStorage serviceSessionStorage
	debouncers            map[string]func(interface{})
}

func NewKeeper(natStatusProvider natStatusProvider, publisher publisher, serviceLister serviceLister, serviceSessionStorage serviceSessionStorage) *Keeper {
	k := &Keeper{
		state: &stateEvent.State{
			NATStatus: stateEvent.NATStatus{
				Status: "not_finished",
			},
		},
		natStatusProvider:     natStatusProvider,
		publisher:             publisher,
		serviceLister:         serviceLister,
		serviceSessionStorage: serviceSessionStorage,
	}
	k.debouncers = map[string]func(interface{}){
		"service": debounce(k.updateServiceState, time.Millisecond*200),
		"nat":     debounce(k.updateNatStatus, time.Millisecond*200),
		"session": debounce(k.updateSessionState, time.Millisecond*200),
	}
	return k
}

func (k *Keeper) updateServiceState(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	k.updateServices()
	go k.publisher.Publish(stateEvent.Topic, *k.state)
}

func (k *Keeper) ConsumeServiceStateEvent(event service.EventPayload) {
	k.debouncers["service"](event)
}

func (k *Keeper) updateServices() {
	services := k.serviceLister.List()
	result := make([]stateEvent.ServiceInfo, len(services))

	i := 0
	for k, v := range services {
		proposal := v.Proposal()
		result[i] = stateEvent.ServiceInfo{
			ID:         string(k),
			ProviderID: proposal.ProviderID,
			Type:       proposal.ServiceType,
			Options:    v.Options(),
			Status:     string(v.State()),
			Proposal:   proposal,
		}
		i++
	}

	k.state.Services = result
}

// ConsumeNATEvent consumes a given NAT event
func (k *Keeper) ConsumeNATEvent(event natEvent.Event) {
	k.debouncers["nat"](event)
}

func (k *Keeper) updateNatStatus(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()

	event, ok := e.(natEvent.Event)
	if !ok {
		return
	}

	k.natStatusProvider.ConsumeNATEvent(event)
	k.state.NATStatus = stateEvent.NATStatus{Status: k.natStatusProvider.Status().Status}

	go k.publisher.Publish(stateEvent.Topic, *k.state)
}

func (k *Keeper) ConsumeSessionEvent(event sessionEvent.Payload) {
	k.debouncers["session"](event)
}

func (k *Keeper) updateSessionState(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()

	sessions := k.serviceSessionStorage.GetAll()
	result := make([]stateEvent.ServiceSession, len(sessions))
	for i := range sessions {
		result[i] = stateEvent.ServiceSession{
			ID:         string(sessions[i].ID),
			ConsumerID: sessions[i].ConsumerID.Address,
			CreatedAt:  sessions[i].CreatedAt,
		}
	}

	k.state.Sessions = result
	go k.publisher.Publish(stateEvent.Topic, *k.state)
}

func debounce(f func(interface{}), d time.Duration) func(interface{}) {
	incoming := make(chan interface{})

	go func() {
		var e interface{}

		t := time.NewTimer(d)
		t.Stop()

		for {
			select {
			case e = <-incoming:
				t.Reset(d)
			case <-t.C:
				go f(e)
			}
		}
	}()

	return func(e interface{}) {
		go func() { incoming <- e }()
	}
}

func (k *Keeper) GetState() event.State {
	k.lock.Lock()
	defer k.lock.Unlock()

	return *k.state
}
