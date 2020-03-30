package config

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
