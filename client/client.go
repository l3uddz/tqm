package client

import (
	"fmt"
	"github.com/antonmedv/expr/vm"
	"strings"
)

func NewClient(clientType string, clientName string, ignorsExpr []*vm.Program, removesExpr []*vm.Program) (Interface, error) {
	switch strings.ToLower(clientType) {
	case "deluge":
		return NewDeluge(clientName, ignorsExpr, removesExpr)
	case "qbittorrent":
		return NewQBittorrent(clientName, ignorsExpr, removesExpr)
	default:
		break
	}

	return nil, fmt.Errorf("client type not implemented: %q", clientType)
}
