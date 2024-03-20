package route

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/data"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/HuolalaTech/page-spy-api/storage"
	"github.com/HuolalaTech/page-spy-api/task"
	"github.com/labstack/gommon/log"
)

type CoreApi struct {
	storage        storage.StorageApi
	data           data.DataApi
	maxSize        int64 // unit byte
	maxLife        int64 // unit Hour
	addressManager *rpc.AddressManager
}

type RcpCoreApi struct {
	core *CoreApi
}

func (c *CoreApi) CreateFileId(md5 string) string {
	return fmt.Sprintf("%s.%s", c.addressManager.GetSelfMachineID(), md5)
}

func (c *CoreApi) IsSelfMachine(machineId string) bool {
	return c.addressManager.GetSelfMachineID() == machineId
}

func (c *CoreApi) GetMachineIdByFileName(name string) (string, error) {
	names := strings.Split(name, ".")
	if len(names) != 2 {
		return "", fmt.Errorf("file name format error")
	}
	return names[0], nil
}

type EmptyReaderClose struct {
	reader io.Reader
}

func (e *EmptyReaderClose) Read(p []byte) (n int, err error) {
	return e.reader.Read(p)
}

func (e *EmptyReaderClose) Close() error {
	return nil
}

func (c *CoreApi) CreateFile(file *storage.LogFile) (*storage.LogFile, error) {
	hash := md5.New()
	reader := io.TeeReader(file.File, hash)

	file.File = &EmptyReaderClose{
		reader: reader,
	}

	md5Sum := hash.Sum(nil)
	md5String := hex.EncodeToString(md5Sum)

	file.FileId = c.CreateFileId(md5String)
	err := c.data.CreateLog(&data.LogData{
		Model: data.Model{
			UpdatedAt: time.Now(),
			CreatedAt: time.Now(),
		},
		FileId: file.FileId,
		Status: data.Created,
		Size:   file.Size,
		Name:   file.Name,
	})

	if err != nil {
		return nil, err
	}

	err = c.storage.SaveLog(file)
	if err != nil {
		return file, c.data.UpdateLogStatus(file.FileId, data.Error)
	}

	return file, c.data.UpdateLogStatus(file.FileId, data.Saved)
}

func (c *CoreApi) GetFileList(size int, page int) (*data.Page[*data.LogData], error) {
	return c.data.FindLogs(size, page)
}

func (c *CoreApi) GetFile(fileId string) (*storage.LogFile, error) {
	return c.storage.GetLog(fileId)
}

func (c *CoreApi) DeleteFile(fileId string) error {
	err := c.storage.RemoveLog(fileId)
	if err != nil {
		return err
	}

	return c.data.DeleteLogByFileId(fileId)
}

func (c *CoreApi) CleanFileByTime() error {
	before := time.Now().Add(-time.Duration(c.maxLife) * 24 * time.Hour)
	logs, err := c.data.FindTimeoutLogs(before, 10)
	if err != nil {
		return err
	}

	if logs == nil || len(logs) <= 0 {
		return nil
	}

	log.Infof("clean file by time %d file timeout", len(logs))
	for _, l := range logs {
		err := c.DeleteFile(l.FileId)
		if err != nil {
			log.Errorf("delete file %s error %s", l.FileId, err.Error())
		}
		log.Infof("clean file %s name %s by time createdAt %s", l.FileId, l.Name, l.CreatedAt.String())
	}

	return nil
}

func (c *CoreApi) CleanFileBySize() error {
	size, err := c.data.CountLogsSize()
	if err != nil {
		return err
	}
	if size < c.maxSize {
		return nil
	}

	log.Infof("clean file by size %dmb > max size %dmb", size/(1024*1024), c.maxSize/(1024*1024))
	logs, err := c.data.FindOldestLogs(10)
	if err != nil {
		return err
	}

	for _, l := range logs {
		err := c.DeleteFile(l.FileId)
		if err != nil {
			log.Errorf("delete file %s error %s", l.FileId, err.Error())
		}
		log.Infof("clean file %s name %s by size", l.FileId, l.Name)
	}

	return nil
}

func (c *CoreApi) CleanFile() error {
	err := c.CleanFileBySize()
	if err != nil {
		log.Errorf("clean file by size error %s", err.Error())
	}
	err = c.CleanFileByTime()
	if err != nil {
		log.Errorf("clean file by time error %s", err.Error())
	}

	return nil
}

func NewCore(config *config.Config, storage storage.StorageApi, taskManager task.TaskManager, data data.DataApi, addressManager *rpc.AddressManager, rpcManager *rpc.RpcManager) (*CoreApi, error) {
	coreApi := &CoreApi{
		storage:        storage,
		data:           data,
		addressManager: addressManager,
		maxSize:        config.MaxLogFileSize * 1024 * 1024,
		maxLife:        config.MaxLogLifeTime,
	}
	err := taskManager.AddTask(task.NewTask("clean_file", 1*time.Hour, coreApi.CleanFile))
	if err != nil {
		log.Errorf("add clean file task error %s", err.Error())
	}

	return coreApi, rpcManager.Regist("CoreApi", NewRpcCore(coreApi))
}

func NewRpcCore(coreApi *CoreApi) *RcpCoreApi {
	return &RcpCoreApi{
		core: coreApi,
	}
}

type FindLogsRequest struct {
	Size int
	Page int
}

func (r *RcpCoreApi) FindLogs(_ *http.Request, req *FindLogsRequest, res *data.Page[*data.LogData]) error {
	page, err := r.core.GetFileList(req.Size, req.Page)
	if err != nil {
		return err
	}
	res.Data = page.Data
	res.Total = page.Total
	return nil
}
