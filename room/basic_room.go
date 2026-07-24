package room

import (
	"sync"

	"github.com/HuolalaTech/page-spy-api/state"
)

func newBasicRoom() *basicRoom {
	return &basicRoom{
		StatusMachine: *state.NewStatusMachine(),
		done:          make(chan struct{}),
	}
}

type basicRoom struct {
	state.StatusMachine
	done      chan struct{}
	closeOnce sync.Once
}

func (r *basicRoom) Done() chan struct{} {
	return r.done
}

func (r *basicRoom) close() bool {
	closed := false
	r.closeOnce.Do(func() {
		closed = true
		r.SetStatus(state.CloseStatus)
		close(r.done)
	})
	return closed
}

func (r *basicRoom) IsClose() bool {
	return r.IsStatus(state.CloseStatus)
}
