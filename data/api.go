package data

import "time"

type DataApi interface {
	CreatePageLog(pageLog *PageLogData) error
	UpdatePageLogFiles(log *PageLogData) error
	FindPageLogByPgeId(pageId string) (*PageLogData, error)
	FindPageLogs(query *FileListQuery) (*Page[*PageLogData], error)

	CreateLog(log *LogData) error
	FindLogs(query *FileListQuery) (*Page[*LogData], error)
	UpdateLogStatus(fileId string, status Status) error
	DeleteLogByFileId(fileId string) error
	FindLogByFileId(fileId string) (*LogData, error)
	FindTimeoutLogs(before time.Time, size int) ([]*LogData, error)
	FindOldestLogs(size int) ([]*LogData, error)
	CountLogsSize() (int64, error)
}
