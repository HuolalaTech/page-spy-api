package task

import (
	"fmt"
	"time"
)

type Task struct {
	name     string
	interval time.Duration
	function func() error
	ticker   *time.Ticker
	done     chan struct{}
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
	close(t.done)
}

func (t *Task) Star() {
	tinker := time.NewTicker(t.interval)
	t.ticker = tinker
	log.Infof("task %s start", t.name)
	go func() {
		for {
			select {
			case <-tinker.C:
				t.run()
			case <-t.done:
				return
			}
		}
	}()
}

func NewTask(name string, interval time.Duration, f func() error) *Task {
	return &Task{
		name:     name,
		interval: interval,
		function: f,
	}
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*Task),
	}
}

type TaskManager struct {
	tasks map[string]*Task
}

func (t *TaskManager) AddTask(task *Task) error {
	findTask, ok := t.tasks[task.name]
	if ok {
		return fmt.Errorf("task %s already exists", findTask.name)
	}

	t.tasks[task.name] = task
	task.Star()
	return nil
}

func (t *TaskManager) Close() error {
	for _, task := range t.tasks {
		task.Close()
	}
	return nil
}
