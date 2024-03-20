package task

type TaskManager struct {
}

type Task interface {
	Run() error
}

func (tm *TaskManager) AddTask() error {
	return nil
}
