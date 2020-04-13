package client

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/dustin/go-humanize"
	"github.com/l3uddz/go-qbittorrent/qbt"
	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/logger"
	"github.com/l3uddz/tqm/sliceutils"
	"github.com/pkg/errors"
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
	client     *qbt.Client

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
		return nil, errors.WithMessagef(err, "failed unmarshalling configuration for client: %s", name)
	}

	// validate config
	if errs := config.ValidateStruct(tc); errs != nil {
		return nil, fmt.Errorf("failed validating client configuration: %v", errs)
	}

	// init client
	tc.client = qbt.NewClient(*tc.Url)

	return &tc, nil
}

/* Interface  */

func (c *QBittorrent) Type() string {
	return c.clientType
}

func (c *QBittorrent) Connect() error {
	// login
	if err := c.client.Login(qbt.LoginOptions{
		Username: *c.User,
		Password: *c.Password,
	}); err != nil {
		return errors.WithMessage(err, "failed logging into client")
	}

	// retrieve & validate api version
	apiVersion, err := c.client.WebAPIVersion()
	if err != nil {
		return errors.WithMessage(err, "failed determining api version")
	} else if apiVersion < 2.2 {
		return fmt.Errorf("unsupported webapi version: %v", apiVersion)
	}

	c.log.Debugf("API Version: %v", apiVersion)
	return nil
}

func (c *QBittorrent) GetTorrents() (map[string]config.Torrent, error) {
	// retrieve torrents from client
	c.log.Tracef("Retrieving torrents...")
	t, err := c.client.Torrents(nil)
	if err != nil {
		return nil, errors.WithMessage(err, "failed retrieving torrents")
	}
	c.log.Tracef("Retrieved %d torrents", len(t))

	// build torrent list
	torrents := make(map[string]config.Torrent)
	for _, t := range t {
		t := t

		// get additional torrent details
		td, err := c.client.Torrent(t.Hash)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed retrieving additional details for torrent: %q", t.Hash)
		}

		ts, err := c.client.TorrentTrackers(t.Hash)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed retrieving tracker details for torrent: %q", t.Hash)
		}

		tf, err := c.client.TorrentFiles(t.Hash)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed retrieving file details for torrent: %q", t.Hash)
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
			trackerStatus = tracker.Msg
			break
		}

		// added time
		addedTime := time.Unix(td.AdditionDate, 0)
		addedTimeSecs := int64(time.Since(addedTime).Seconds())

		// torrent files
		var files []string
		for _, f := range tf {
			files = append(files, filepath.Join(t.SavePath, f.Name))
		}

		// create torrent
		torrent := config.Torrent{
			Hash:            t.Hash,
			Name:            t.Name,
			Path:            t.SavePath,
			TotalBytes:      t.Size,
			DownloadedBytes: t.Size - t.AmountLeft,
			State:           t.State,
			Files:           files,
			Downloaded:      t.AmountLeft == 0,
			Seeding:         sliceutils.StringSliceContains([]string{"uploading", "stalledUP"}, t.State, true),
			Ratio:           td.ShareRatio,
			AddedSeconds:    addedTimeSecs,
			AddedHours:      float32(addedTimeSecs) / 60 / 60,
			AddedDays:       float32(addedTimeSecs) / 60 / 60 / 24,
			SeedingSeconds:  td.SeedingTime,
			SeedingHours:    float32(td.SeedingTime) / 60 / 60,
			SeedingDays:     float32(td.SeedingTime) / 60 / 60 / 24,
			Label:           t.Category,
			Seeds:           td.SeedsTotal,
			Peers:           td.PeersTotal,
			// free space
			FreeSpaceGB:  c.freeSpaceGB,
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
	if _, err := c.client.Pause([]string{hash}); err != nil {
		return false, errors.Wrapf(err, "failed pausing torrent: %q", hash)
	}

	time.Sleep(1 * time.Second)

	// resume torrent
	if _, err := c.client.Resume([]string{hash}); err != nil {
		return false, errors.Wrapf(err, "failed resuming torrent: %q", hash)
	}

	// sleep before removing torrent
	time.Sleep(2 * time.Second)

	// remove
	return c.client.Delete([]string{hash}, deleteData)
}

func (c *QBittorrent) GetCurrentFreeSpace(path string) (int64, error) {
	// get current main stats
	data, err := c.client.SyncMainData()
	if err != nil {
		return 0, errors.Wrapf(err, "failed retrieving maindata")
	}

	// set internal free size
	c.freeSpaceGB = float64(data.ServerState.FreeSpaceOnDisk) / humanize.GiByte
	c.freeSpaceSet = true

	return data.ServerState.FreeSpaceOnDisk, nil
}

func (c *QBittorrent) GetFreeSpace() float64 {
	return c.freeSpaceGB
}

/* Filters */

func (c *QBittorrent) ShouldIgnore(t *config.Torrent) (bool, error) {
	for _, expression := range c.ignoresExpr {
		result, err := expr.Run(expression, t)
		if err != nil {
			return true, errors.Wrap(err, "failed checking ignore expression")
		}

		expResult, ok := result.(bool)
		if !ok {
			return true, errors.New("failed type asserting ignore expression result")
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
			return false, errors.Wrap(err, "failed checking remove expression")
		}

		expResult, ok := result.(bool)
		if !ok {
			return false, errors.New("failed type asserting remove expression result")
		}

		if expResult {
			return true, nil
		}
	}

	return false, nil
}
