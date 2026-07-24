package task

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestTaskCloseIsIdempotent(t *testing.T) {
	var calls atomic.Int64
	task := NewTask("test", time.Millisecond, func() error {
		calls.Add(1)
		return nil
	})

	task.Start()
	task.Close()
	task.Close()

	select {
	case <-task.done:
	default:
		t.Fatal("task done channel was not closed")
	}
}

func TestTaskClosedBeforeStartDoesNotRun(t *testing.T) {
	var calls atomic.Int64
	task := NewTask("test", time.Millisecond, func() error {
		calls.Add(1)
		return nil
	})

	task.Close()
	task.Start()
	time.Sleep(5 * time.Millisecond)
	if got := calls.Load(); got != 0 {
		t.Fatalf("closed task ran %d times", got)
	}
}

func TestTaskManagerCloseIsIdempotent(t *testing.T) {
	manager := NewTaskManager()
	if err := manager.AddTask(NewTask("test", time.Hour, func() error { return nil })); err != nil {
		t.Fatalf("add task: %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("close manager: %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("close manager again: %v", err)
	}
	if err := manager.AddTask(NewTask("late", time.Hour, func() error { return nil })); err == nil {
		t.Fatal("closed manager accepted a new task")
	}
}

func TestTaskManagerRejectsInvalidTasks(t *testing.T) {
	manager := NewTaskManager()
	if err := manager.AddTask(nil); err == nil {
		t.Fatal("task manager accepted a nil task")
	}
	if err := manager.AddTask(NewTask("nil-function", time.Hour, nil)); err == nil {
		t.Fatal("task manager accepted a task with a nil function")
	}
}
