package route

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/HuolalaTech/page-spy-api/data"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/HuolalaTech/page-spy-api/storage"
)

type CoreApi struct {
	storage        storage.StorageApi
	data           data.DataApi
	addressManager *rpc.AddressManager
}

func (c *CoreApi) CreateFileId(md5 string) string {
	return fmt.Sprintf("%s.%s", c.addressManager.GetSelfMachineID(), md5)
}

func (c *CoreApi) GetMachineIdByFileName(name string) (string, error) {
	names := strings.Split(name, ".")
	if len(names) != 2 {
		return "", fmt.Errorf("file name format error")
	}
	return names[1], nil
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
	err := c.storage.SaveLog(file)
	if err != nil {
		return file, err
	}

	return file, nil
}

func (c *CoreApi) GetFile(fileId string) (*storage.LogFile, error) {
	return c.storage.GetLog(fileId)
}

func (c *CoreApi) DeleteFile(fileId string) error {
	return c.storage.RemoveLog(fileId)
}

func (c *CoreApi) DeleteTimeoutFile() error {
	return nil
}

func (c *CoreApi) DeleteOldestFile() error {
	return nil
}

func NewCore(storage storage.StorageApi, data data.DataApi, addressManager *rpc.AddressManager) *CoreApi {
	return &CoreApi{storage: storage, data: data, addressManager: addressManager}
}
