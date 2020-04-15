package client

import (
	"github.com/l3uddz/tqm/config"
)

type Interface interface {
	// general
	Type() string
	Connect() error
	GetTorrents() (map[string]config.Torrent, error)
	RemoveTorrent(string, bool) (bool, error)
	GetCurrentFreeSpace(string) (int64, error)
	AddFreeSpace(int64)
	GetFreeSpace() float64

	// filters
	ShouldIgnore(*config.Torrent) (bool, error)
	ShouldRemove(*config.Torrent) (bool, error)
}
