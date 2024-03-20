package data

import "time"

type DataApi interface {
	CreateLog(log *LogData) error
	FindLogs(size int64, page int64) (*Page[*LogData], error)
	UpdateLogStatus(fileId string, status Status) error
	DeleteLogByName(name string) error
	FindLogByName(name string) (*LogData, error)
	FindTimeoutLogs(before time.Time) ([]*LogData, error)
	FindOldestLogs() ([]*LogData, error)
	CountLogsSize() (int64, error)
}
