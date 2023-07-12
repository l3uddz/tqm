package config

import (
	"fmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/l3uddz/tqm/logger"
	"github.com/l3uddz/tqm/stringutils"
	"github.com/l3uddz/tqm/tracker"
)

type Configuration struct {
	Clients               map[string]map[string]interface{}
	Filters               map[string]FilterConfiguration
	Trackers              tracker.Config
	TorrentRetentionLimit int
}

/* Vars */

var (
	cfgPath = ""

	Delimiter = "."
	Config    *Configuration
	K         = koanf.New(Delimiter)

	// Internal
	log = logger.GetLogger("cfg")
)

/* Public */

func Init(configFilePath string) error {
	// set package variables
	cfgPath = configFilePath

	// load config
	if err := K.Load(file.Provider(configFilePath), yaml.Parser()); err != nil {
		return fmt.Errorf("load: %w", err)
	}

	// Initialize Config
	Config = &Configuration{}

	// Load torrent_retention_limit separately.
	Config.TorrentRetentionLimit = K.Int("torrent_retention_limit")

	// unmarshal config
	if err := K.Unmarshal("", &Config); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return nil
}

func ShowUsing() {
	log.Infof("Using %s = %q", stringutils.LeftJust("CONFIG", " ", 10), cfgPath)

}
