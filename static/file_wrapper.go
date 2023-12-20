package static

import (
	"io"
	"io/fs"
)

var (
	_ fs.File     = (*fileWrapper)(nil)
	_ fs.FileInfo = (*fileInfoWrapper)(nil)
)

type fileWrapper struct {
	buf io.Reader
	fi  fs.FileInfo
}

func (f *fileWrapper) Stat() (fs.FileInfo, error) {
	return f.fi, nil
}

func (f *fileWrapper) Read(bytes []byte) (int, error) {
	return f.buf.Read(bytes)
}

func (f *fileWrapper) Close() error {
	return nil
}

type fileInfoWrapper struct {
	fs.FileInfo
	size int
}

func (f fileInfoWrapper) Size() int64 {
	return int64(f.size)
}
