package mcp

import (
	"net/url"
	"strconv"
	"strings"

	eventApi "github.com/HuolalaTech/page-spy-api/api/event"
)

func parseAddress(id string) (*eventApi.Address, error) {
	return eventApi.NewAddressFromID(strings.TrimSpace(id))
}

func parseIntQuery(u *url.URL, key string, def int) int {
	v := u.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

