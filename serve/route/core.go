package route

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/data"
	"github.com/HuolalaTech/page-spy-api/logger"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/HuolalaTech/page-spy-api/storage"
	"github.com/HuolalaTech/page-spy-api/task"
)

var log = logger.Log().WithField("module", "core")

type CoreApi struct {
	rpcManager     *rpc.RpcManager
	storage        storage.StorageApi
	data           data.DataApi
	maxSizeOfByte  int64 // unit byte
	maxLifeOfHour  int64 // unit Hour
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
	hash := md5.Sum(file.UpdateFile)
	md5String := hex.EncodeToString(hash[:])
	file.FileId = c.CreateFileId(md5String)

	err := c.storage.SaveLog(file)
	if err != nil {
		return file, err
	}

	err = c.data.CreateLog(&data.LogData{
		Model: data.Model{
			UpdatedAt: time.Now(),
			CreatedAt: time.Now(),
		},
		Tags:   file.Tags,
		FileId: file.FileId,
		Status: data.Saved,
		Size:   file.Size,
		Name:   file.Name,
	})

	if err != nil {
		return nil, err
	}

	return file, err
}

func (c *CoreApi) getFileList(query *data.FileListQuery) (*data.Page[*data.LogData], error) {
	return c.data.FindLogs(query)
}

func (c *CoreApi) GetFileList(query *data.FileListQuery) (*data.Page[*data.LogData], error) {
	res := &data.Page[*data.LogData]{}
	err := rpc.CallAllClient(c.rpcManager, context.Background(), "CoreApi.FindLogs", query, res)

	if err != nil {
		return nil, err
	}

	res.Desc()
	return res, nil
}

func (c *CoreApi) GetFile(fileId string) (*storage.LogFile, error) {
	fileData, err := c.data.FindLogByFileId(fileId)
	if err != nil {
		return nil, err
	}
	if fileData == nil {
		return nil, fmt.Errorf("file %s not found", fileId)
	}

	logFile, err := c.storage.GetLog(fileId)
	if err != nil {
		return nil, err
	}

	logFile.Name = fileData.Name
	return logFile, nil
}

func (c *CoreApi) DeleteFile(fileId string) error {
	err := c.storage.RemoveLog(fileId)
	if err != nil {
		return err
	}

	return c.data.DeleteLogByFileId(fileId)
}

func (c *CoreApi) CleanFileByTime() error {
	before := time.Now().Add(-time.Duration(c.maxLifeOfHour) * time.Hour)
	logs, err := c.data.FindTimeoutLogs(before, 10)
	if err != nil {
		return err
	}

	if logs == nil || len(logs) <= 0 {
		return nil
	}

	log.Infof("clean file by time %d file timeout before %s", len(logs), before.String())
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

	if size < c.maxSizeOfByte {
		return nil
	}

	log.Infof("clean file by size %dmb > max size %dmb", size/(1024*1024), c.maxSizeOfByte/(1024*1024))
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

func NewCore(config *config.Config, storage storage.StorageApi, taskManager *task.TaskManager, data data.DataApi, addressManager *rpc.AddressManager, rpcManager *rpc.RpcManager) (*CoreApi, error) {
	maxLogFileSizeOfMb := config.GetMaxLogFileSizeOfMB()

	maxLifeOfHour := config.GetMaxLogLifeTimeOfHour()

	coreApi := &CoreApi{
		storage:        storage,
		rpcManager:     rpcManager,
		data:           data,
		addressManager: addressManager,
		maxSizeOfByte:  maxLogFileSizeOfMb * 1024 * 1024,
		maxLifeOfHour:  maxLifeOfHour,
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

func (r *RcpCoreApi) FindLogs(_ *http.Request, req *data.FileListQuery, res *data.Page[*data.LogData]) error {
	page, err := r.core.getFileList(req)
	if err != nil {
		return err
	}
	res.Data = page.Data
	res.Total = page.Total
	return nil
}
