package data

import "time"

type DataApi interface {
	CreateLog(log *LogData) error
	FindLogs(size int, page int) (*Page[*LogData], error)
	UpdateLogStatus(fileId string, status Status) error
	DeleteLogByFileId(fileId string) error
	FindLogByFileId(fileId string) (*LogData, error)
	FindTimeoutLogs(before time.Time, size int) ([]*LogData, error)
	FindOldestLogs(size int) ([]*LogData, error)
	CountLogsSize() (int64, error)
}
