package data

import "time"

type DataApi interface {
	CreateLogGroup(logGroup *LogGroup) error
	UpdateLogGroup(logGroup *LogGroup) error
	FindLogGroup(groupId string) (*LogGroup, error)
	FindLogGroups(query *FileListQuery) (*Page[*LogGroup], error)
	DeleteLogGroupByGroupId(groupId string) error

	CountLogsGroup(tagKey string) ([]LogGroupResult, error)
	CreateLog(log *LogData) error
	FindLogs(query *FileListQuery) (*Page[*LogData], error)
	UpdateLogStatus(fileId string, status Status) error
	DeleteLogByFileId(fileId string) error
	FindLogByFileId(fileId string) (*LogData, error)
	FindTimeoutLogs(before time.Time, size int) ([]*LogData, error)
	FindOldestLogs(size int) ([]*LogData, error)
	CountLogsSize() (int64, error)
}