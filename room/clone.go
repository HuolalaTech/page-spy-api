package room

import (
	"github.com/HuolalaTech/page-spy-api/api/event"
	roomApi "github.com/HuolalaTech/page-spy-api/api/room"
)

func cloneAddress(address *event.Address) *event.Address {
	if address == nil {
		return nil
	}

	cloned := *address
	return &cloned
}

func cloneConnection(connection *roomApi.Connection) *roomApi.Connection {
	if connection == nil {
		return nil
	}

	cloned := *connection
	cloned.Address = cloneAddress(connection.Address)
	return &cloned
}

func cloneConnections(connections []*roomApi.Connection) []*roomApi.Connection {
	cloned := make([]*roomApi.Connection, 0, len(connections))
	for _, connection := range connections {
		cloned = append(cloned, cloneConnection(connection))
	}
	return cloned
}

func cloneTags(tags map[string]string) map[string]string {
	if tags == nil {
		return nil
	}

	cloned := make(map[string]string, len(tags))
	for key, value := range tags {
		cloned[key] = value
	}
	return cloned
}

func cloneRoomInfo(info *roomApi.Info) *roomApi.Info {
	if info == nil {
		return nil
	}

	cloned := *info
	cloned.Address = cloneAddress(info.Address)
	cloned.Tags = cloneTags(info.Tags)
	cloned.Connections = cloneConnections(info.Connections)
	return &cloned
}
