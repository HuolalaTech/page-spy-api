package room

import (
	"encoding/json"
	"fmt"

	"github.com/HuolalaTech/page-spy-api/api/event"
	"github.com/HuolalaTech/page-spy-api/api/room"
)

func roomMessageToPackage(msg *room.Message, from *event.Address) (*event.Package, error) {
	bs, err := json.Marshal(msg.Content)
	if err != nil {
		return nil, fmt.Errorf("room message to message error %w", err)
	}

	return &event.Package{
		From:       from,
		RoutingKey: msg.Type,
		Content:    bs,
	}, nil
}

func packageToRoomMessage(pkg *event.Package) (*room.Message, error) {
	content := room.NewMessageContent(pkg.RoutingKey)
	err := json.Unmarshal(pkg.Content, content)
	if err != nil {
		return nil, fmt.Errorf("raw message to message error %w", err)
	}

	return &room.Message{
		Type:    pkg.RoutingKey,
		Content: content,
	}, nil
}
