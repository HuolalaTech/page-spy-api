package data

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/HuolalaTech/page-spy-api/config"
	selfLogger "github.com/HuolalaTech/page-spy-api/logger"
	"github.com/HuolalaTech/page-spy-api/storage"
	"github.com/HuolalaTech/page-spy-api/task"
	"github.com/HuolalaTech/page-spy-api/util"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type Data struct {
	db *gorm.DB
}

func getLocalDataFilePath() string {
	if util.FileExists("data.db") {
		return "data.db"
	}

	if util.FileExists("data/data.db") {
		return "data/data.db"
	}

	return "data/data.db"
}

func initDataFilePath() (string, error) {
	fileInfo, err := os.Stat("data")
	if (err != nil && os.IsNotExist(err)) || (err == nil && !fileInfo.IsDir()) {
		err := os.Mkdir("data", 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create data directory")
		}
	}

	return getLocalDataFilePath(), nil
}

var logger = selfLogger.Log().WithField("module", "database")

func InitData(config *gorm.Config) (*Data, error) {
	dataPath, err := initDataFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to init data path")
	}

	logger.Infof("init database with file %s", dataPath)
	db, err := gorm.Open(sqlite.Open(dataPath), config)

	if err != nil {
		return nil, fmt.Errorf("failed to connect database")
	}

	if err := db.AutoMigrate(&LogData{}, &LogGroup{}, &Tag{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate database")
	}

	return &Data{db: db}, nil
}

func NewData(config *config.Config, taskManager *task.TaskManager, storage storage.StorageApi) (DataApi, error) {
	if config.IsRemoteStorage() {
		logger.Infof("init database with remote storage")
		err := loadData(config, storage)
		if err != nil {
			logger.Infof("load remote data error %s", err.Error())
			return nil, err
		}
		logger.Infof("load remote data success")

		err = taskManager.AddTask(task.NewTask("sync_data_file", 5*time.Minute, syncData(config, storage)))
		if err != nil {
			logger.Errorf("add sync data file task error %s", err.Error())
			return nil, err
		}
	}

	logLevel := gormLogger.Silent
	if config.Debug {
		logLevel = gormLogger.Info
	}

	c := &gorm.Config{
		Logger: gormLogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormLogger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logLevel,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	}

	return InitData(c)
}

func loadData(config *config.Config, s storage.StorageApi) error {
	filePath := getLocalDataFilePath()
	if util.FileExists(filePath) {
		logger.Infof("load data already exists")
		return nil
	}

	remotePath := path.Join(config.StorageConfig.BaseDir, filePath)
	remoteStorage := s.(*storage.RemoteApi)
	exist, err := remoteStorage.Exist(remotePath)
	if err != nil {
		return fmt.Errorf("failed to head remote data file %s", err.Error())
	}
	if !exist {
		logger.Infof("load data remote data not exists")
		return nil
	}

	body, _, err := remoteStorage.Get(remotePath)
	if err != nil {
		return fmt.Errorf("failed to get remote data file %s", err.Error())
	}

	if !exist {
		return nil
	}

	_, err = initDataFilePath()
	if err != nil {
		return err
	}

	defer body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, body); err != nil {
		return err
	}

	return nil
}

func syncData(config *config.Config, s storage.StorageApi) func() error {
	return func() error {
		filePath := getLocalDataFilePath()
		if !util.FileExists(filePath) {
			return nil
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			logger.Errorf("Failed to read file: %v", err)
			return err
		}

		remotePath := path.Join(config.StorageConfig.BaseDir, filePath)
		remoteStorage := s.(*storage.RemoteApi)
		err = remoteStorage.Save(remotePath, bytes.NewReader(content))
		if err != nil {
			return err
		}

		return nil
	}
}

func (d *Data) UpdateLogGroup(groupLog *LogGroup) error {
	result := d.db.Model(groupLog).Updates(&LogGroup{
		Size: groupLog.Size,
		Logs: groupLog.Logs,
	})

	return result.Error
}

func (d *Data) CreateLogGroup(groupLog *LogGroup) error {
	result := d.db.Create(groupLog)
	return result.Error
}

func (d *Data) FindLogGroup(groupId string) (*LogGroup, error) {
	logGroup := &LogGroup{}
	result := d.db.Where("group_id = ?", groupId).Preload("Logs").First(logGroup)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return logGroup, result.Error
}

func (d *Data) FindLogGroups(query *FileListQuery) (*Page[*LogGroup], error) {
	if query.Size <= 0 {
		return nil, fmt.Errorf("size should be greater than 0")
	}

	if query.Page <= 0 {
		return nil, fmt.Errorf("page should be greater than 0")
	}

	var logGroups []*LogGroup
	offset := query.GetOffset()
	result := query.getLogGroupDB(d.db).Offset(offset).Limit(query.Size).Find(&logGroups)
	if result.Error != nil {
		return nil, result.Error
	}

	var total int64
	result = query.getLogGroupDB(d.db).Model(&LogGroup{}).Count(&total)
	if result.Error != nil {
		return nil, result.Error
	}

	return &Page[*LogGroup]{
		Data:  logGroups,
		Total: total,
	}, nil
}

func (d *Data) DeleteLogGroupByGroupId(groupId string) error {
	logGroup, err := d.FindLogGroup(groupId)
	if err != nil {
		return err
	}

	result := d.db.Delete(logGroup)
	return result.Error
}

func (d *Data) CreateLog(log *LogData) error {
	findFile, err := d.FindLogByFileId(log.FileId)

	if err != nil {
		return err
	}

	if findFile != nil {
		return nil
	}

	result := d.db.Create(log)
	return result.Error
}

type PageQuery struct {
	Size int
	Page int
}

func (p *PageQuery) GetOffset() int {
	return (p.Page - 1) * p.Size
}

type FileListQuery struct {
	PageQuery
	From *int64
	To   *int64
	Tags []*storage.Tag
}

func (f *FileListQuery) GetFrom() *time.Time {
	if f.From == nil {
		return nil
	}

	from := time.Unix(*f.From, 0)

	return &from
}

func (f *FileListQuery) GetTo() *time.Time {
	if f.To == nil {
		return nil
	}

	to := time.Unix(*f.To, 0)

	return &to
}

func (query *FileListQuery) getLogGroupDB(db *gorm.DB) *gorm.DB {
	q := db
	if query.Tags != nil && len(query.Tags) > 0 {
		for i, tag := range query.Tags {
			logTagName := fmt.Sprintf("log_group_tag%d", i)
			tagName := fmt.Sprintf("tag%d", i)
			q = q.Joins(fmt.Sprintf("join log_group_tags as %s on %s.log_group_id = log_groups.id", logTagName, logTagName)).
				Joins(fmt.Sprintf("join tags as %s on %s.id = %s.tag_id and %s.key = ? and %s.value like ?", tagName, tagName, logTagName, tagName, tagName), tag.Key, "%"+tag.Value+"%")
		}
	}

	from := query.GetFrom()
	if from != nil {
		q = q.Where("log_groups.created_at > ?", from)
	}

	to := query.GetTo()
	if to != nil {
		q = q.Where("log_groups.created_at < ?", to)
	}

	return q.Preload("Tags").Preload("Logs").Order("log_groups.created_at desc")
}

func (query *FileListQuery) getLogDB(db *gorm.DB) *gorm.DB {
	q := db

	if query.Tags != nil && len(query.Tags) > 0 {
		for i, tag := range query.Tags {
			logTagName := fmt.Sprintf("log_tag%d", i)
			tagName := fmt.Sprintf("tag%d", i)
			q = q.Joins(fmt.Sprintf("join log_tags as %s on %s.log_data_id = log_data.id", logTagName, logTagName)).
				Joins(fmt.Sprintf("join tags as %s on %s.id = %s.tag_id and %s.key = ? and %s.value like ?", tagName, tagName, logTagName, tagName, tagName), tag.Key, "%"+tag.Value+"%")
		}
	}

	from := query.GetFrom()
	if from != nil {
		q = q.Where("log_data.created_at > ?", from)
	}

	to := query.GetTo()
	if to != nil {
		q = q.Where("log_data.created_at < ?", to)
	}

	return q.Preload("Tags").Order("log_data.created_at desc")
}

func (d *Data) FindLogs(query *FileListQuery) (*Page[*LogData], error) {
	if query.Size <= 0 {
		return nil, fmt.Errorf("size should be greater than 0")
	}

	if query.Page <= 0 {
		return nil, fmt.Errorf("page should be greater than 0")
	}

	var logs []*LogData
	offset := query.GetOffset()
	result := query.getLogDB(d.db).Offset(offset).Limit(query.Size).Find(&logs)
	if result.Error != nil {
		return nil, result.Error
	}

	var total int64
	result = query.getLogDB(d.db).Model(&LogData{}).Count(&total)
	if result.Error != nil {
		return nil, result.Error
	}

	return &Page[*LogData]{
		Data:  logs,
		Total: total,
	}, nil
}

func (d *Data) UpdateLogStatus(fileId string, status Status) error {
	result := d.db.Model(&LogData{}).Where("file_id = ?", fileId).Update("status", status)
	return result.Error
}

func (d *Data) DeleteLogByFileId(fileId string) error {
	result := d.db.Where("file_id = ?", fileId).Delete(&LogData{})
	return result.Error
}

func (d *Data) FindLogByFileId(FileId string) (*LogData, error) {
	log := &LogData{}
	result := d.db.Where("file_id = ?", FileId).Where("status = ?", Saved).First(log)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return log, result.Error
}

func (d *Data) FindTimeoutLogs(before time.Time, size int) ([]*LogData, error) {
	var logs []*LogData
	result := d.db.Where("created_at < ?", before).Limit(size).Order("created_at desc").Find(&logs)
	return logs, result.Error
}

func (d *Data) FindOldestLogs(size int) ([]*LogData, error) {
	var logs []*LogData
	result := d.db.Limit(size).Order("created_at asc").Find(&logs)
	return logs, result.Error
}

func (d *Data) FindShouldDeleteLogs(size int) ([]*LogData, error) {
	var logs []*LogData
	status := []Status{
		Error,
		Created,
		Unknown,
	}

	result := d.db.Limit(size).
		Where("created_at < ?", time.Now().Add(-time.Hour*1)).
		Where("status in ?", status).Find(&logs)
	return logs, result.Error
}

type Sum struct {
	Total int64
}

func (d *Data) CountLogsSize() (int64, error) {
	sum := &Sum{}
	result := d.db.Model(&LogData{}).
		Where("status = ?", Saved).
		Select("sum(size) as total").Scan(sum)
	if result.Error != nil {
		return 0, result.Error
	}

	return sum.Total, nil
}
