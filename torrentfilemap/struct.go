package torrentfilemap

import "github.com/l3uddz/tqm/config"

type TorrentFileMap struct {
	torrentFileMap map[string]map[string]config.Torrent
}
