package static

import (
	"io/fs"
	"os"
)

type FallbackFS struct {
	FS       fs.FS
	Fallback string
}

func (f *FallbackFS) Open(name string) (fs.File, error) {
	file, err := f.FS.Open(name)
	if os.IsNotExist(err) {
		return f.FS.Open(f.Fallback)
	}

	return file, err
}
