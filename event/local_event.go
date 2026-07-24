package event

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/HuolalaTech/page-spy-api/api/event"
	"github.com/HuolalaTech/page-spy-api/rpc"
)

func NewLocalEventEmitter(addressManager *rpc.AddressManager, rpcManager *rpc.RpcManager) event.EventEmitter {
	return &LocalEventEmitter{
		listeners:      make(map[string][]event.Listener),
		rpcManager:     rpcManager,
		addressManager: addressManager,
	}
}

type LocalEventEmitter struct {
	rpcManager     *rpc.RpcManager
	listeners      map[string][]event.Listener
	addressManager *rpc.AddressManager
	rwLock         sync.RWMutex
}

func sameListener(left event.Listener, right event.Listener) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	leftValue := reflect.ValueOf(left)
	rightValue := reflect.ValueOf(right)
	if leftValue.Type() != rightValue.Type() || !leftValue.Type().Comparable() {
		return false
	}
	return leftValue.Interface() == rightValue.Interface()
}

func (e *LocalEventEmitter) addListener(address *event.Address, listener event.Listener) {
	if address == nil || listener == nil {
		return
	}

	e.rwLock.Lock()
	defer e.rwLock.Unlock()
	list := e.listeners[address.ID]
	if list == nil {
		list = []event.Listener{}
	}
	for _, l := range list {
		if sameListener(l, listener) {
			return
		}
	}

	list = append(list, listener)
	e.listeners[address.ID] = list
}

func (e *LocalEventEmitter) RemoveListener(address *event.Address, listener event.Listener) {
	if address == nil || listener == nil {
		return
	}

	e.rwLock.Lock()
	defer e.rwLock.Unlock()
	list := e.listeners[address.ID]
	if list == nil {
		return
	}

	newList := []event.Listener{}
	for _, l := range list {
		if !sameListener(l, listener) {
			newList = append(newList, l)
		}
	}

	if len(newList) == 0 {
		delete(e.listeners, address.ID)
		return
	}
	e.listeners[address.ID] = newList
}

func (e *LocalEventEmitter) getListeners(address *event.Address) []event.Listener {
	if address == nil {
		return []event.Listener{}
	}

	e.rwLock.RLock()
	defer e.rwLock.RUnlock()
	list := e.listeners[address.ID]
	if list == nil {
		list = []event.Listener{}
	}

	return append([]event.Listener(nil), list...)
}

func (e *LocalEventEmitter) emitRemote(ctx context.Context, address *event.Address, pkg *event.Package) error {
	if e.rpcManager == nil {
		return fmt.Errorf("rpc manager is nil")
	}

	req := NewRpcEventEmitterRequest()
	req.Address = address
	req.Package = pkg
	res := NewRpcEventEmitterResponse()
	client := e.rpcManager.GetRpcByAddress(address)
	if client == nil {
		return fmt.Errorf("rpc client %s not found", address.MachineID)
	}

	err := client.Call(ctx, "RpcEventEmitter.Emit", req, res)
	if err != nil {
		return err
	}

	return res.GetError()
}

func (e *LocalEventEmitter) EmitLocal(ctx context.Context, address *event.Address, pkg *event.Package) error {
	if address == nil {
		return fmt.Errorf("emit local message address is nil")
	}
	if pkg == nil {
		return fmt.Errorf("emit local message package is nil")
	}

	list := e.getListeners(address)
	if len(list) <= 0 {
		return fmt.Errorf("Emit message no Listeners %s", pkg.Content)
	}

	for _, l := range list {
		if !l.IsClose() {
			l.Listen(ctx, pkg)
		} else {
			e.RemoveListener(address, l)
		}
	}

	return nil
}

func (e *LocalEventEmitter) Emit(ctx context.Context, address *event.Address, msg *event.Package) error {
	if address == nil {
		return fmt.Errorf("emit message address is nil")
	}
	if msg == nil {
		return fmt.Errorf("emit message package is nil")
	}
	if e.addressManager == nil {
		return fmt.Errorf("address manager is nil")
	}

	if e.addressManager.IsSelfMachineAddress(address) {
		return e.EmitLocal(ctx, address, msg)
	}

	return e.emitRemote(ctx, address, msg)
}

func (e *LocalEventEmitter) Listen(address *event.Address, listener event.Listener) {
	e.addListener(address, listener)
}

func (e *LocalEventEmitter) Close() error {
	return nil
}
