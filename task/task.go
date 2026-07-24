package task

import (
	"fmt"
	"sync"
	"time"
)

type Task struct {
	name      string
	interval  time.Duration
	function  func() error
	ticker    *time.Ticker
	done      chan struct{}
	mu        sync.Mutex
	startOnce sync.Once
	closeOnce sync.Once
}

func (t *Task) run() {
	defer func() {
		if err := recover(); err != nil {
			log.Infof("task %s panic %s", t.name, err)
		}
	}()

	err := t.function()
	if err != nil {
		log.Infof("task %s error %s", t.name, err.Error())
	}
}

func (t *Task) Close() {
	t.closeOnce.Do(func() {
		close(t.done)
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.ticker != nil {
			t.ticker.Stop()
		}
	})
}

func (t *Task) Start() {
	t.startOnce.Do(func() {
		t.mu.Lock()
		select {
		case <-t.done:
			t.mu.Unlock()
			return
		default:
		}

		ticker := time.NewTicker(t.interval)
		t.ticker = ticker
		t.mu.Unlock()

		log.Infof("task %s start", t.name)
		go func() {
			for {
				select {
				case <-ticker.C:
					t.run()
				case <-t.done:
					return
				}
			}
		}()
	})
}

// Star 保留用于兼容现有调用方，请使用 Start。
func (t *Task) Star() {
	t.Start()
}

func NewTask(name string, interval time.Duration, f func() error) *Task {
	return &Task{
		name:     name,
		interval: interval,
		function: f,
		done:     make(chan struct{}),
	}
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*Task),
	}
}

type TaskManager struct {
	tasks  map[string]*Task
	mu     sync.Mutex
	closed bool
}

func (t *TaskManager) AddTask(task *Task) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("task manager is closed")
	}
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if task.function == nil {
		return fmt.Errorf("task %s function is nil", task.name)
	}

	findTask, ok := t.tasks[task.name]
	if ok {
		return fmt.Errorf("task %s already exists", findTask.name)
	}

	t.tasks[task.name] = task
	task.Start()
	return nil
}

func (t *TaskManager) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true

	tasks := make([]*Task, 0, len(t.tasks))
	for _, task := range t.tasks {
		tasks = append(tasks, task)
	}
	t.mu.Unlock()

	for _, task := range tasks {
		task.Close()
	}
	return nil
}
