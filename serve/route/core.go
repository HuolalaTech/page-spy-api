package route

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HuolalaTech/page-spy-api/data"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/HuolalaTech/page-spy-api/storage"
)

type CoreApi struct {
	storage        storage.StorageApi
	data           data.DataApi
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

func (c *CoreApi) CleanFile() error {
	return nil
}

func NewCore(storage storage.StorageApi, data data.DataApi, addressManager *rpc.AddressManager, rpcManager *rpc.RpcManager) (*CoreApi, error) {
	coreApi := &CoreApi{storage: storage, data: data, addressManager: addressManager}
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
