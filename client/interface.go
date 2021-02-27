package client

import (
	"github.com/l3uddz/tqm/config"
)

type Interface interface {
	Type() string
	Connect() error
	GetTorrents() (map[string]config.Torrent, error)
	RemoveTorrent(string, bool) (bool, error)
	SetTorrentLabel(string, string) error
	GetCurrentFreeSpace(string) (int64, error)
	AddFreeSpace(int64)
	GetFreeSpace() float64

	ShouldIgnore(*config.Torrent) (bool, error)
	ShouldRemove(*config.Torrent) (bool, error)
}
