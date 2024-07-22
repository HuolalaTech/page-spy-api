package data

import "time"

type DataApi interface {
	CreateGroupLog(groupLog *LogGroup) error
	UpdateGroupLog(groupLog *LogGroup) error
	FindGroupLog(groupId string) (*LogGroup, error)
	FindGroupLogs(query *FileListQuery) (*Page[*LogGroup], error)
	DeleteLogGroupByGroupId(groupId string) error

	CreateLog(log *LogData) error
	FindLogs(query *FileListQuery) (*Page[*LogData], error)
	UpdateLogStatus(fileId string, status Status) error
	DeleteLogByFileId(fileId string) error
	FindLogByFileId(fileId string) (*LogData, error)
	FindTimeoutLogs(before time.Time, size int) ([]*LogData, error)
	FindOldestLogs(size int) ([]*LogData, error)
	CountLogsSize() (int64, error)
}
