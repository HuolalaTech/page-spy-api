package room

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	eventApi "github.com/HuolalaTech/page-spy-api/api/event"
	roomApi "github.com/HuolalaTech/page-spy-api/api/room"
)

type testEventEmitter struct{}

func (testEventEmitter) Emit(context.Context, *eventApi.Address, *eventApi.Package) error {
	return nil
}

func (testEventEmitter) EmitLocal(context.Context, *eventApi.Address, *eventApi.Package) error {
	return nil
}

func (testEventEmitter) Listen(*eventApi.Address, eventApi.Listener) {}

func (testEventEmitter) RemoveListener(*eventApi.Address, eventApi.Listener) {}

func (testEventEmitter) Close() error { return nil }

type testRPCRoom struct {
	info *roomApi.Info
}

func (r *testRPCRoom) GetRoomAddress() *eventApi.Address {
	return r.info.Address
}

func (r *testRPCRoom) GetInfo() *roomApi.Info {
	return r.info
}

func (r *testRPCRoom) UpdateInfo(info *roomApi.Info) {
	r.info = info
}

func testRoomInfo() *roomApi.Info {
	return roomApi.NewRoomInfo(
		"room",
		"",
		false,
		map[string]string{"env": "test"},
		"group",
		&eventApi.Address{ID: "room.local", LocalID: "room", MachineID: "local"},
	)
}

func TestBasicRoomCloseIsIdempotent(t *testing.T) {
	basic := newBasicRoom()
	var closedCount atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if basic.close() {
				closedCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := closedCount.Load(); got != 1 {
		t.Fatalf("room closed %d times, want 1", got)
	}
	select {
	case <-basic.Done():
	default:
		t.Fatal("room done channel was not closed")
	}
}

func TestLocalRoomInfoIsSnapshot(t *testing.T) {
	created, err := NewLocalRoom(testRoomInfo(), testEventEmitter{}, nil)
	if err != nil {
		t.Fatalf("create local room: %v", err)
	}
	local := created.(*localRoom)

	snapshot := local.GetInfo()
	snapshot.Name = "changed"
	snapshot.Tags["env"] = "changed"
	snapshot.Address.ID = "changed.local"

	current := local.GetInfo()
	if current.Name != "room" {
		t.Fatalf("room name was mutated through snapshot: %q", current.Name)
	}
	if current.Tags["env"] != "test" {
		t.Fatalf("room tags were mutated through snapshot: %q", current.Tags["env"])
	}
	if current.Address.ID != "room.local" {
		t.Fatalf("room address was mutated through snapshot: %q", current.Address.ID)
	}
}

func TestNewLocalRoomPreservesOptionInitialization(t *testing.T) {
	opt := testRoomInfo()
	opt.Connections = nil
	opt.CreatedAt = time.Time{}
	opt.ActiveAt = time.Time{}

	if _, err := NewLocalRoom(opt, testEventEmitter{}, nil); err != nil {
		t.Fatalf("create local room: %v", err)
	}

	if opt.Connections == nil {
		t.Fatal("NewLocalRoom should preserve the original behavior of initializing Connections")
	}
	if len(opt.Connections) != 0 {
		t.Fatalf("expected no initialized connections, got %d", len(opt.Connections))
	}
	if opt.CreatedAt.IsZero() {
		t.Fatal("NewLocalRoom should preserve the original behavior of initializing CreatedAt")
	}
	if opt.ActiveAt.IsZero() {
		t.Fatal("NewLocalRoom should preserve the original behavior of initializing ActiveAt")
	}
}

func TestLocalRoomJSONUsesInfoSnapshot(t *testing.T) {
	created, err := NewLocalRoom(testRoomInfo(), testEventEmitter{}, nil)
	if err != nil {
		t.Fatalf("create local room: %v", err)
	}

	data, err := json.Marshal(created)
	if err != nil {
		t.Fatalf("marshal local room: %v", err)
	}

	var decoded localRoom
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal local room: %v", err)
	}
	if decoded.GetInfo() == nil || decoded.GetInfo().Address.ID != "room.local" {
		t.Fatalf("unexpected decoded room info: %#v", decoded.GetInfo())
	}
}

func TestLocalRoomStateConcurrentAccess(t *testing.T) {
	created, err := NewLocalRoom(testRoomInfo(), testEventEmitter{}, nil)
	if err != nil {
		t.Fatalf("create local room: %v", err)
	}
	local := created.(*localRoom)

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(3)
		go func(index int) {
			defer wg.Done()
			local.UpdateInfo(&roomApi.Info{BasicInfo: roomApi.BasicInfo{
				Name: "updated",
				Tags: map[string]string{"iteration": "value"},
			}})
		}(i)
		go func() {
			defer wg.Done()
			local.Ping()
		}()
		go func() {
			defer wg.Done()
			_ = local.GetInfo()
			_ = local.GetRoomUsers()
			_ = local.GetTags()
			_, _ = local.ShouldRemove()
		}()
	}
	wg.Wait()
}

func TestRemoteRoomRejectsMissingTargetAddress(t *testing.T) {
	info := testRoomInfo()
	connection := &roomApi.Connection{
		Address: &eventApi.Address{ID: "connection.local", LocalID: "connection", MachineID: "local"},
	}
	remote, err := NewRemoteRoom(connection, info, testEventEmitter{}, &testRPCRoom{info: info})
	if err != nil {
		t.Fatalf("create remote room: %v", err)
	}

	err = remote.SendMessage(context.Background(), &roomApi.Message{
		Type: roomApi.MessageType,
		Content: &roomApi.MessageMessageContent{
			To: &roomApi.Connection{},
		},
	})
	if err == nil {
		t.Fatal("message with missing target address was accepted")
	}
}
