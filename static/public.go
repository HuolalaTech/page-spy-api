package static

import (
	"bytes"
	"io"
	"io/fs"
	"os"
)

type FallbackFS struct {
	FS         fs.FS
	Fallback   string
	PublicPath string
	BaseAPIURL string
}

var (
	publicPathPlaceholder = []byte("__PAGE_SPY_PUBLIC_PATH__")
	baseAPIURLPlaceholder = []byte("__PAGE_SPY_BASE_API_URL__")
)

func NewFallbackFS(fs fs.FS, fallback string, publicPath, baseAPIURL string) *FallbackFS {
	return &FallbackFS{
		FS:         fs,
		Fallback:   fallback,
		PublicPath: publicPath,
		BaseAPIURL: baseAPIURL,
	}
}

func (f *FallbackFS) Open(name string) (fs.File, error) {
	file, err := f.open(name)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return file, nil
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}

	content = bytes.ReplaceAll(content, publicPathPlaceholder, []byte(f.PublicPath))
	content = bytes.ReplaceAll(content, baseAPIURLPlaceholder, []byte(f.BaseAPIURL))
	return &fileWrapper{
		buf: bytes.NewReader(content),
		fi: fileInfoWrapper{
			FileInfo: fi,
			size:     len(content),
		},
	}, nil
}

func (f *FallbackFS) open(name string) (fs.File, error) {
	file, err := f.FS.Open(name)
	if os.IsNotExist(err) {
		return f.FS.Open(f.Fallback)
	}

	return file, err
}
