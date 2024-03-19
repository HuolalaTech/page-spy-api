package data

import "time"

type DataApi interface {
	CreateLog(log *LogData) error
	UpdateLog(log *LogData) error
	FindLogByName(name string) (*LogData, error)
	FindTimeoutLogs(before time.Time) ([]*LogData, error)
	FindOldestLogs() ([]*LogData, error)
	CountLogsSize() (int64, error)
}
