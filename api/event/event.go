package event

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Address struct {
	ID        string `json:"id"`
	MachineID string `json:"machineId"`
	LocalID   string `json:"localId"`
}

func (a *Address) ToString() string {
	return a.ID
}

func (a *Address) GetMachineID() string {
	return a.MachineID
}

func (a *Address) Equal(compare *Address) bool {
	if compare == nil {
		return false
	}

	return a.ID == compare.ID
}

func NewAddressFromID(id string) (*Address, error) {
	words := strings.Split(id, ".")
	if len(words) != 2 {
		return nil, fmt.Errorf("address id %s is an invalid format", id)
	}

	return &Address{
		ID:        id,
		LocalID:   words[0],
		MachineID: words[1],
	}, nil
}

func (a *Address) MarshalJSON() ([]byte, error) {
	return []byte("\"" + a.ToString() + "\""), nil
}

func (a *Address) UnmarshalJSON(data []byte) error {
	var id string
	if err := json.Unmarshal(data, &id); err != nil {
		return err
	}

	address, err := NewAddressFromID(id)
	if err != nil {
		return err
	}

	*a = *address
	return nil
}

type Package struct {
	From       *Address        `json:"from"`
	CreatedAt  int64           `json:"createdAt"`
	RequestId  string          `json:"requestId"`
	RoutingKey string          `json:"routingKey"`
	Content    json.RawMessage `json:"content"`
}

type Listener interface {
	Listen(ctx context.Context, pkg *Package)
	IsClose() bool
	Close(ctx context.Context) error
}

type EventEmitter interface {
	Emit(ctx context.Context, address *Address, pkg *Package) error
	EmitLocal(ctx context.Context, address *Address, pkg *Package) error
	Listen(address *Address, listener Listener)
	RemoveListener(address *Address, listener Listener)
	Close() error
}
