package client

import (
	"fmt"
	"github.com/KnutZuidema/go-qbittorrent"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/dustin/go-humanize"
	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/logger"
	"github.com/l3uddz/tqm/sliceutils"
	"github.com/l3uddz/tqm/stringutils"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
	"time"
)

/* Struct */

type QBittorrent struct {
	Url      *string `validate:"required"`
	User     *string `validate:"required"`
	Password *string `validate:"required"`

	// internal
	log        *logrus.Entry
	clientType string
	client     *qbittorrent.Client

	// set by cmd handler
	freeSpaceGB  float64
	freeSpaceSet bool

	// internal compiled filters
	ignoresExpr []*vm.Program
	removesExpr []*vm.Program
}

/* Initializer */

func NewQBittorrent(name string, ignoresExpr []*vm.Program, removesExpr []*vm.Program) (Interface, error) {
	tc := QBittorrent{
		log:         logger.GetLogger(name),
		clientType:  "qBittorrent",
		ignoresExpr: ignoresExpr,
		removesExpr: removesExpr,
	}

	// load config
	if err := config.K.Unmarshal(fmt.Sprintf("clients%s%s", config.Delimiter, name), &tc); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// validate config
	if errs := config.ValidateStruct(tc); errs != nil {
		return nil, fmt.Errorf("validate config: %v", errs)
	}

	// init client
	tc.client = qbittorrent.NewClient(*tc.Url, nil)

	return &tc, nil
}

/* Interface  */

func (c *QBittorrent) Type() string {
	return c.clientType
}

func (c *QBittorrent) Connect() error {
	// login
	if err := c.client.Login(*c.User, *c.Password); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	// retrieve & validate api version
	apiVersion, err := c.client.Application.GetAPIVersion()
	if err != nil {
		return fmt.Errorf("get api version: %w", err)
	} else if stringutils.Atof64(apiVersion, 0.0) < 2.2 {
		return fmt.Errorf("unsupported webapi version: %v", apiVersion)
	}

	c.log.Debugf("API Version: %v", apiVersion)
	return nil
}

func (c *QBittorrent) GetTorrents() (map[string]config.Torrent, error) {
	// retrieve torrents from client
	c.log.Tracef("Retrieving torrents...")
	t, err := c.client.Torrent.GetList(nil)
	if err != nil {
		return nil, fmt.Errorf("get torrents: %w", err)
	}
	c.log.Tracef("Retrieved %d torrents", len(t))

	// build torrent list
	torrents := make(map[string]config.Torrent)
	for _, t := range t {
		t := t

		// get additional torrent details
		td, err := c.client.Torrent.GetProperties(t.Hash)
		if err != nil {
			return nil, fmt.Errorf("get torrent properties: %v: %w", t.Hash, err)
		}

		ts, err := c.client.Torrent.GetTrackers(t.Hash)
		if err != nil {
			return nil, fmt.Errorf("get torrent trackers: %v: %w", t.Hash, err)
		}

		tf, err := c.client.Torrent.GetContents(t.Hash)
		if err != nil {
			return nil, fmt.Errorf("get torrent files: %v: %w", t.Hash, err)
		}

		// parse tracker details
		trackerName := ""
		trackerStatus := ""

		for _, tracker := range ts {
			// skip disabled trackers
			if strings.Contains(tracker.URL, "[DHT]") || strings.Contains(tracker.URL, "[LSD]") ||
				strings.Contains(tracker.URL, "[PeX]") {
				continue
			}

			// use status of first enabled tracker
			trackerName = parseTrackerDomain(tracker.URL)
			trackerStatus = tracker.Message
			break
		}

		// added time
		addedTime := td.AdditionDate
		addedTimeSecs := int64(time.Since(addedTime).Seconds())

		// torrent files
		var files []string
		for _, f := range tf {
			files = append(files, filepath.Join(td.SavePath, f.Name))
		}

		// create torrent
		torrent := config.Torrent{
			Hash:            t.Hash,
			Name:            t.Name,
			Path:            td.SavePath,
			TotalBytes:      int64(t.Size),
			DownloadedBytes: int64(td.TotalDownloaded),
			State:           string(t.State),
			Files:           files,
			Downloaded:      td.TotalDownloaded >= t.Size,
			Seeding:         sliceutils.StringSliceContains([]string{"uploading", "stalledUP"}, string(t.State), true),
			Ratio:           float32(td.ShareRatio),
			AddedSeconds:    addedTimeSecs,
			AddedHours:      float32(addedTimeSecs) / 60 / 60,
			AddedDays:       float32(addedTimeSecs) / 60 / 60 / 24,
			SeedingSeconds:  int64(td.SeedingTime),
			SeedingHours:    float32(td.SeedingTime) / 60 / 60,
			SeedingDays:     float32(td.SeedingTime) / 60 / 60 / 24,
			Label:           t.Category,
			Seeds:           int64(td.SeedsTotal),
			Peers:           int64(td.PeersTotal),
			// free space
			FreeSpaceGB:  c.GetFreeSpace,
			FreeSpaceSet: c.freeSpaceSet,
			// tracker
			TrackerName:   trackerName,
			TrackerStatus: trackerStatus,
		}

		torrents[t.Hash] = torrent
	}

	return torrents, nil
}

func (c *QBittorrent) RemoveTorrent(hash string, deleteData bool) (bool, error) {
	// pause torrent
	if err := c.client.Torrent.StopTorrents([]string{hash}); err != nil {
		return false, fmt.Errorf("pause torrent: %v: %w", hash, err)
	}

	time.Sleep(1 * time.Second)

	// resume torrent
	if err := c.client.Torrent.ResumeTorrents([]string{hash}); err != nil {
		return false, fmt.Errorf("resume torrent: %v: %w", hash, err)
	}

	// sleep before re-announcing torrent
	time.Sleep(2 * time.Second)

	if err := c.client.Torrent.ReannounceTorrents([]string{hash}); err != nil {
		return false, fmt.Errorf("re-announce torrent: %v: %w", hash, err)
	}

	// sleep before removing torrent
	time.Sleep(2 * time.Second)

	// remove
	if err := c.client.Torrent.DeleteTorrents([]string{hash}, deleteData); err != nil {
		return false, fmt.Errorf("delete torrent: %v: %w", hash, err)
	}

	return true, nil
}

func (c *QBittorrent) SetTorrentLabel(hash string, label string) error {
	// set label
	if err := c.client.Torrent.SetCategories([]string{hash}, label); err != nil {
		return fmt.Errorf("set torrent label: %v: %w", label, err)
	}

	return nil
}

func (c *QBittorrent) GetCurrentFreeSpace(path string) (int64, error) {
	// get current main stats
	data, err := c.client.Sync.GetMainData(0)
	if err != nil {
		return 0, fmt.Errorf("get main data: %w", err)
	}

	// set internal free size
	c.freeSpaceGB = float64(data.ServerState.FreeSpaceOnDisk) / humanize.GiByte
	c.freeSpaceSet = true

	return int64(data.ServerState.FreeSpaceOnDisk), nil
}

func (c *QBittorrent) AddFreeSpace(bytes int64) {
	c.freeSpaceGB += float64(bytes) / humanize.GiByte
}

func (c *QBittorrent) GetFreeSpace() float64 {
	return c.freeSpaceGB
}

/* Filters */

func (c *QBittorrent) ShouldIgnore(t *config.Torrent) (bool, error) {
	for _, expression := range c.ignoresExpr {
		result, err := expr.Run(expression, t)
		if err != nil {
			return true, fmt.Errorf("check ignore expression: %w", err)
		}

		expResult, ok := result.(bool)
		if !ok {
			return true, fmt.Errorf("type assert ignore expression result: %w", err)
		}

		if expResult {
			return true, nil
		}
	}

	return false, nil
}

func (c *QBittorrent) ShouldRemove(t *config.Torrent) (bool, error) {
	for _, expression := range c.removesExpr {
		result, err := expr.Run(expression, t)
		if err != nil {
			return false, fmt.Errorf("check remeove expression: %w", err)
		}

		expResult, ok := result.(bool)
		if !ok {
			return false, fmt.Errorf("type assert remove expression result: %w", err)
		}

		if expResult {
			return true, nil
		}
	}

	return false, nil
}
