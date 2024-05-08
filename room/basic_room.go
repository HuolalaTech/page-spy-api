package room

import (
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
	done chan struct{}
}

func (r *basicRoom) Done() chan struct{} {
	return r.done
}

func (r *basicRoom) close() error {
	if r.IsStatus(state.CloseStatus) {
		return nil
	}

	r.SetStatus(state.CloseStatus)
	close(r.done)
	return nil
}

func (r *basicRoom) IsClose() bool {
	return r.IsStatus(state.CloseStatus)
}
