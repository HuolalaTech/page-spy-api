package event

import (
	"context"
	"net/http"
	"time"

	"github.com/HuolalaTech/page-spy-api/api/event"
	"github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/rpc"
)

type RpcEventEmitter struct {
	localEventEmitter event.EventEmitter
}

type RpcEventEmitterRequest struct {
	ContextTimeout int
	Address        *event.Address
	Package        *event.Package
}

func NewRpcEventEmitterRequest() *RpcEventEmitterRequest {
	return &RpcEventEmitterRequest{
		ContextTimeout: 5,
	}
}

type RpcEventEmitterResponse struct {
	room.BasicRpcResponse
}

func NewRpcEventEmitterResponse() *RpcEventEmitterResponse {
	return &RpcEventEmitterResponse{}
}

func NewRpcEventEmitter(localEventEmitter event.EventEmitter, rpcManager *rpc.RpcManager) (*RpcEventEmitter, error) {
	event := &RpcEventEmitter{
		localEventEmitter: localEventEmitter,
	}

	return event, rpcManager.Regist("RpcEventEmitter", event)
}

func (e *RpcEventEmitter) Emit(r *http.Request, req *RpcEventEmitterRequest, res *RpcEventEmitterResponse) error {
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(req.ContextTimeout)*time.Second)
	defer cancel()

	err := e.localEventEmitter.EmitLocal(ctx, req.Address, req.Package)
	res.SetError(err)
	return nil
}
