package state

import "sync/atomic"

type Status int

const (
	InitStatus    Status = 1
	RunningStatus Status = 2
	CloseStatus   Status = 3
	ErrorStatus   Status = 4
)

func NewStatusMachine() *StatusMachine {
	sm := StatusMachine{}
	sm.SetStatus(InitStatus)
	return &sm
}

type StatusMachine struct {
	status atomic.Value
}

func (e *StatusMachine) SetStatus(s Status) {
	e.status.Store(s)
}

func (e *StatusMachine) IsStatus(s Status) bool {
	return e.status.Load().(Status) == s
}
