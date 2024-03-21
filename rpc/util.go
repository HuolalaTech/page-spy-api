package rpc

import (
	"context"
	"fmt"
)

type MergeResult interface {
	Merge(result MergeResult) error
	New() MergeResult
}

func CallAllClient[T MergeResult](r *RpcManager, ctx context.Context, method string, req any, res T) error {
	if len(r.rpcList) == 0 {
		return fmt.Errorf("rpc client list is empty")
	}

	for _, r := range r.rpcList {
		tmp := res.New()
		err := r.Call(ctx, method, req, tmp)
		if err != nil {
			return err
		}

		err = res.Merge(tmp)
		if err != nil {
			return err
		}
	}

	return nil
}
