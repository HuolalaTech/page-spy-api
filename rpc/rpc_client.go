package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/rpc"
	"sync"
	"time"

	"github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/state"
	req "github.com/imroc/req/v3"
)

type RpcClient struct {
	state.StatusMachine
	client  *rpc.Client
	lock    sync.RWMutex
	address string
	err     error
	id      int64
}

func NewRpcClient(address string) *RpcClient {
	req.SetTimeout(4 * time.Second)
	return &RpcClient{
		StatusMachine: *state.NewStatusMachine(),
		address:       address,
	}
}

func (r *RpcClient) getId() int64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.id = r.id + 1
	return r.id
}

func (r *RpcClient) initClient() {
	if !r.IsStatus(state.InitStatus) && !r.IsStatus(state.ErrorStatus) {
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.IsStatus(state.InitStatus) && !r.IsStatus(state.ErrorStatus) {
		return
	}

	client, err := rpc.Dial("tcp", r.address)
	if err != nil {
		r.err = err
		r.SetStatus(state.ErrorStatus)
		return
	}

	r.SetStatus(state.RunningStatus)
	r.client = client
	r.err = nil
}

func (r *RpcClient) GetClient() (*rpc.Client, error) {
	r.initClient()
	if r.err != nil && r.IsStatus(state.ErrorStatus) {
		return nil, r.err
	}

	return r.client, nil
}

type Result struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
	Id     int64       `json:"id"`
}

func (r *RpcClient) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	id := r.getId()
	body := map[string]interface{}{
		"method": serviceMethod,
		"params": []interface{}{args},
		"id":     id,
	}

	result := &Result{
		Result: reply,
		Error:  "",
		Id:     id,
	}

	resp, err := req.R().SetContext(ctx).SetBody(body).SetResult(result).Post(fmt.Sprintf("http://%s/rpc", r.address))
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		bs := resp.Bytes()
		err = json.Unmarshal(bs, result)
		if err != nil {
			return fmt.Errorf("request status %s error %s", resp.Status, string(bs))
		}

		return fmt.Errorf(result.Error)
	}

	basicRes, ok := reply.(room.BasicRpcResponseInterface)
	if ok && basicRes.GetError() != nil {
		return basicRes.GetError()
	}

	return nil
}
