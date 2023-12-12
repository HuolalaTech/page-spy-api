package event

import (
	"context"
	"fmt"
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

func (e *LocalEventEmitter) addListener(address *event.Address, listener event.Listener) {
	e.rwLock.Lock()
	defer e.rwLock.Unlock()
	list := e.listeners[address.ID]
	if list == nil {
		list = []event.Listener{}
	}
	for _, l := range list {
		if &l == &listener {
			return
		}
	}

	list = append(list, listener)
	e.listeners[address.ID] = list
}

func (e *LocalEventEmitter) RemoveListener(address *event.Address, listener event.Listener) {
	e.rwLock.Lock()
	defer e.rwLock.Unlock()
	list := e.listeners[address.ID]
	if list == nil {
		return
	}

	newList := []event.Listener{}
	for _, l := range list {
		if l != listener {
			newList = append(newList, l)
		}
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

	return list
}

func (e *LocalEventEmitter) emitRemote(ctx context.Context, address *event.Address, pkg *event.Package) error {
	req := NewRpcEventEmitterRequest()
	req.Address = address
	req.Package = pkg
	res := NewRpcEventEmitterResponse()
	err := e.rpcManager.GetRpcByAddress(address).Call(ctx, "RpcEventEmitter.Emit", req, res)
	if err != nil {
		return err
	}

	return res.GetError()
}

func (e *LocalEventEmitter) EmitLocal(ctx context.Context, address *event.Address, pkg *event.Package) error {
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
