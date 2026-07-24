package route

import (
	"testing"

	"github.com/HuolalaTech/page-spy-api/data"
)

type emptyLogGroupData struct {
	data.DataApi
}

func (emptyLogGroupData) FindLogGroup(string) (*data.LogGroup, error) {
	return nil, nil
}

func TestListFilesInMissingGroupReturnsError(t *testing.T) {
	core := &CoreApi{data: emptyLogGroupData{}}
	if _, err := core.ListFilesInGroup("missing"); err == nil {
		t.Fatal("missing log group did not return an error")
	}
}
