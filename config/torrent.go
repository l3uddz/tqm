package config

import "strings"

var (
	unregisteredStatuses = []string{
		"unregistered torrent",
		"torrent is not authorized for use on this tracker",
		"torrent is not found",
		"not registered with this tracker",
		"torrent not found",
		"unregistered torrent",
	}
)

type Torrent struct {
	// torrent
	Hash            string
	Name            string
	Path            string
	TotalBytes      int64
	DownloadedBytes int64
	State           string
	Files           []string
	Downloaded      bool
	Seeding         bool
	Ratio           float32
	AddedSeconds    int64
	AddedHours      float32
	AddedDays       float32
	SeedingSeconds  int64
	SeedingHours    float32
	SeedingDays     float32
	Label           string
	Seeds           int64
	Peers           int64

	// tracker
	TrackerName   string
	TrackerStatus string
}

func (t *Torrent) IsUnregistered() bool {
	for _, v := range unregisteredStatuses {
		// unregistered tracker status found?
		if strings.Contains(strings.ToLower(t.TrackerStatus), v) {
			return true
		}
	}

	return false
}
