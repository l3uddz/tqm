package client

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/l3uddz/tqm/expression"
	"path"
	"time"

	"github.com/antonmedv/expr"
	delugeclient "github.com/gdm85/go-libdeluge"
	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/logger"
	"github.com/sirupsen/logrus"
)

/* Struct */

type Deluge struct {
	Host     *string `validate:"required"`
	Port     *uint   `validate:"required"`
	Login    *string `validate:"required"`
	Password *string `validate:"required"`
	V2       bool

	// internal
	log        *logrus.Entry
	clientType string
	client     *delugeclient.LabelPlugin
	client1    *delugeclient.Client
	client2    *delugeclient.ClientV2

	// set by cmd handler
	freeSpaceGB  float64
	freeSpaceSet bool

	// internal compiled filters
	exp *expression.Expressions
}

/* Initializer */

func NewDeluge(name string, exp *expression.Expressions) (Interface, error) {
	tc := Deluge{
		log:        logger.GetLogger(name),
		clientType: "Deluge",
		exp:        exp,
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
	settings := delugeclient.Settings{
		Hostname: *tc.Host,
		Port:     *tc.Port,
		Login:    *tc.Login,
		Password: *tc.Password,
	}

	if tc.V2 {
		tc.client2 = delugeclient.NewV2(settings)
	} else {
		tc.client1 = delugeclient.NewV1(settings)
	}

	return &tc, nil
}

/* Interface  */

func (c *Deluge) Type() string {
	return c.clientType
}

func (c *Deluge) Connect() error {
	var err error

	// connect to deluge daemon
	c.log.Tracef("Connecting to %s:%d", *c.Host, *c.Port)

	if c.V2 {
		err = c.client2.Connect()
	} else {
		err = c.client1.Connect()

	}

	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	// retrieve & set common label client
	var lc *delugeclient.LabelPlugin

	if c.V2 {
		lc, err = c.client2.LabelPlugin()
	} else {
		lc, err = c.client1.LabelPlugin()
	}

	if err != nil {
		return fmt.Errorf("get label plugin: %w", err)
	}

	// retrieve daemon version
	daemonVersion, err := lc.DaemonVersion()
	if err != nil {
		return fmt.Errorf("get daemon version: %w", err)
	}
	c.log.Debugf("Daemon Version: %v", daemonVersion)

	c.client = lc
	return nil
}

func (c *Deluge) GetTorrents() (map[string]config.Torrent, error) {
	// retrieve torrents from client
	c.log.Tracef("Retrieving torrents...")
	t, err := c.client.TorrentsStatus(delugeclient.StateUnspecified, nil)
	if err != nil {
		return nil, fmt.Errorf("get torrents: %w", err)
	}
	c.log.Tracef("Retrieved %d torrents", len(t))

	// retrieve torrent labels
	labels, err := c.client.GetTorrentsLabels(delugeclient.StateUnspecified, nil)
	if err != nil {
		return nil, fmt.Errorf("get torrent labels: %w", err)
	}
	c.log.Tracef("Retrieved labels for %d torrents", len(labels))

	// build torrent list
	torrents := make(map[string]config.Torrent)
	for h, t := range t {
		h := h
		t := t

		// build files slice
		var files []string
		for _, f := range t.Files {
			files = append(files, path.Join(t.DownloadLocation, f.Path))
		}

		// get torrent label
		label := ""
		if l, ok := labels[h]; ok {
			label = l
		}

		// create torrent object
		torrent := config.Torrent{
			// torrent
			Hash:            h,
			Name:            t.Name,
			Path:            t.DownloadLocation,
			TotalBytes:      t.TotalSize,
			DownloadedBytes: t.TotalDone,
			State:           t.State,
			Files:           files,
			Downloaded:      t.TotalDone == t.TotalSize,
			Seeding:         t.IsSeed,
			Ratio:           t.Ratio,
			AddedSeconds:    t.ActiveTime,
			AddedHours:      float32(t.ActiveTime) / 60 / 60,
			AddedDays:       float32(t.ActiveTime) / 60 / 60 / 24,
			SeedingSeconds:  t.SeedingTime,
			SeedingHours:    float32(t.SeedingTime) / 60 / 60,
			SeedingDays:     float32(t.SeedingTime) / 60 / 60 / 24,
			Label:           label,
			Seeds:           t.TotalSeeds,
			Peers:           t.TotalPeers,
			// free space
			FreeSpaceGB:  c.GetFreeSpace,
			FreeSpaceSet: c.freeSpaceSet,
			// tracker
			TrackerName:   t.TrackerHost,
			TrackerStatus: t.TrackerStatus,
		}

		torrents[h] = torrent
	}

	return torrents, nil
}

func (c *Deluge) RemoveTorrent(hash string, deleteData bool) (bool, error) {
	// pause torrent
	if err := c.client.PauseTorrents(hash); err != nil {
		return false, fmt.Errorf("pause torrent: %v: %w", hash, err)
	}

	time.Sleep(1 * time.Second)

	// resume torrent
	if err := c.client.ResumeTorrents(hash); err != nil {
		return false, fmt.Errorf("resume torrent: %v: %w", hash, err)
	}

	// sleep before re-announcing torrent
	time.Sleep(2 * time.Second)

	// re-announce torrent
	if err := c.client.ForceReannounce([]string{hash}); err != nil {
		return false, fmt.Errorf("re-announce torrent: %v: %w", hash, err)
	}

	// sleep before removing torrent
	time.Sleep(2 * time.Second)

	// remove
	if ok, err := c.client.RemoveTorrent(hash, deleteData); err != nil {
		return false, fmt.Errorf("remove torrent: %v: %w", hash, err)
	} else if !ok {
		return false, fmt.Errorf("remove torrent: %v", hash)
	}

	return true, nil
}

func (c *Deluge) SetTorrentLabel(hash string, label string) error {
	// set label
	if err := c.client.SetTorrentLabel(hash, label); err != nil {
		return fmt.Errorf("set torrent label: %v: %w", label, err)
	}

	return nil
}

func (c *Deluge) GetCurrentFreeSpace(path string) (int64, error) {
	// get free disk space
	space, err := c.client.GetFreeSpace(path)
	if err != nil {
		return 0, fmt.Errorf("get free disk space: %v: %w", path, err)
	}

	// set internal free size
	c.freeSpaceGB = float64(space) / humanize.GiByte
	c.freeSpaceSet = true

	return space, nil
}

func (c *Deluge) AddFreeSpace(bytes int64) {
	c.freeSpaceGB += float64(bytes) / humanize.GiByte
}

func (c *Deluge) GetFreeSpace() float64 {
	return c.freeSpaceGB
}

/* Filters */

func (c *Deluge) ShouldIgnore(t *config.Torrent) (bool, error) {
	for _, expression := range c.exp.Ignores {
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

func (c *Deluge) ShouldRemove(t *config.Torrent) (bool, error) {
	for _, expression := range c.exp.Removes {
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
