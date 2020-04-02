package client

import (
	"github.com/l3uddz/tqm/config"
)

type Interface interface {
	// general
	Type() string
	Connect() error
	GetTorrents() (map[string]config.Torrent, error)
	RemoveTorrent(hash string, deleteData bool) (bool, error)
	GetCurrentFreeSpace(path string) (int64, error)
	GetFreeSpace() float64

	// filters
	ShouldIgnore(*config.Torrent) (bool, error)
	ShouldRemove(*config.Torrent) (bool, error)
}
