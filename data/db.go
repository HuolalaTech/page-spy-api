package data

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Data struct {
	db *gorm.DB
}

func NewData(config *config.Config) (DataApi, error) {
	c := &gorm.Config{}
	if config.Debug {
		c.Logger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				ParameterizedQueries:      false,
				Colorful:                  false,
			},
		)
	}

	db, err := gorm.Open(sqlite.Open("data.db"), c)

	if err != nil {
		return nil, fmt.Errorf("failed to connect database")
	}

	if err := db.AutoMigrate(&LogData{}, &Tag{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate database")
	}

	return &Data{db: db}, nil
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
	Tags []*Tag
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

func (query *FileListQuery) getDB(db *gorm.DB) *gorm.DB {
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
	result := query.getDB(d.db).Offset(offset).Limit(query.Size).Find(&logs)
	if result.Error != nil {
		return nil, result.Error
	}

	var total int64
	result = query.getDB(d.db).Model(&LogData{}).Count(&total)
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
