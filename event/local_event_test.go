package event

import (
	"context"
	"sync/atomic"
	"testing"

	eventApi "github.com/HuolalaTech/page-spy-api/api/event"
)

type testListener struct {
	closed atomic.Bool
}

func (*testListener) Listen(context.Context, *eventApi.Package) {}

func (l *testListener) IsClose() bool {
	return l.closed.Load()
}

func (l *testListener) Close(context.Context, string) error {
	l.closed.Store(true)
	return nil
}

func TestListenerRegistrationIsDeduplicated(t *testing.T) {
	emitter := NewLocalEventEmitter(nil, nil).(*LocalEventEmitter)
	address := &eventApi.Address{ID: "listener.local", LocalID: "listener", MachineID: "local"}
	listener := &testListener{}

	emitter.Listen(address, listener)
	emitter.Listen(address, listener)

	listeners := emitter.getListeners(address)
	if len(listeners) != 1 {
		t.Fatalf("registered %d listeners, want 1", len(listeners))
	}

	listeners[0] = nil
	if got := len(emitter.getListeners(address)); got != 1 {
		t.Fatalf("mutating returned listener slice changed emitter state: %d", got)
	}
}

func TestRemoveLastListenerDeletesEntry(t *testing.T) {
	emitter := NewLocalEventEmitter(nil, nil).(*LocalEventEmitter)
	address := &eventApi.Address{ID: "listener.local", LocalID: "listener", MachineID: "local"}
	listener := &testListener{}

	emitter.Listen(address, listener)
	emitter.RemoveListener(address, listener)
	if _, exists := emitter.listeners[address.ID]; exists {
		t.Fatal("empty listener entry was retained")
	}
}

func TestLocalEventRejectsInvalidMessages(t *testing.T) {
	emitter := NewLocalEventEmitter(nil, nil).(*LocalEventEmitter)
	address := &eventApi.Address{ID: "listener.local", LocalID: "listener", MachineID: "local"}

	if err := emitter.EmitLocal(context.Background(), nil, &eventApi.Package{}); err == nil {
		t.Fatal("EmitLocal accepted a nil address")
	}
	if err := emitter.EmitLocal(context.Background(), address, nil); err == nil {
		t.Fatal("EmitLocal accepted a nil package")
	}
	if err := emitter.Emit(context.Background(), address, &eventApi.Package{}); err == nil {
		t.Fatal("Emit accepted a missing address manager")
	}
}
