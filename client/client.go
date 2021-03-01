package client

import (
	"fmt"
	"github.com/l3uddz/tqm/expression"
	"strings"
)

func NewClient(clientType string, clientName string, exp *expression.Expressions) (Interface, error) {
	switch strings.ToLower(clientType) {
	case "deluge":
		return NewDeluge(clientName, exp)
	case "qbittorrent":
		return NewQBittorrent(clientName, exp)
	default:
		break
	}

	return nil, fmt.Errorf("client type not implemented: %q", clientType)
}
