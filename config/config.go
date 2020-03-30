package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/l3uddz/tqm/logger"
	"github.com/l3uddz/tqm/stringutils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	yaml2 "gopkg.in/yaml.v2"
)

type Configuration struct {
	Clients map[string]map[string]interface{}
	Filters map[string]FilterConfiguration
}

/* Vars */

var (
	cfgPath = ""

	// Config exports the config object
	Delimiter = "."
	Config    *Configuration
	K         = koanf.New(Delimiter)

	// Internal
	log          = logger.GetLogger("cfg")
	newOptionLen = 0
)

/* Public */

func (cfg Configuration) ToJsonString() (string, error) {
	c := viper.AllSettings()
	bs, err := json.MarshalIndent(c, "", "  ")
	return string(bs), err
}

func Init(configFilePath string) error {
	// set package variables
	cfgPath = configFilePath

	// load config file
	if err := K.Load(file.Provider(configFilePath), yaml.Parser()); err == nil {
		if err := K.Unmarshal("", &Config); err != nil {
			return errors.WithMessage(err, "failed unmarshalling configuration file")
		}
	}

	// set config defaults
	if err := setConfigDefaults(true); err != nil {
		log.WithError(err).Fatal("Failed setting configuration defaults...")
	}

	return nil
}

func ShowUsing() {
	log.Infof("Using %s = %q", stringutils.LeftJust("CONFIG", " ", 10), cfgPath)

}

/* Private */

func setConfigDefault(key string, value interface{}, check bool) int {
	if check {
		if K.Exists(key) {
			return 0
		}

		// determine padding to use for new key
		if keyLen := len(key); (keyLen + 2) > newOptionLen {
			newOptionLen = keyLen + 2
		}

		log.Warnf("New config option: %s = %+v", stringutils.LeftJust(fmt.Sprintf("%q", key),
			" ", newOptionLen), value)
	}

	if err := K.Load(confmap.Provider(map[string]interface{}{key: value}, Delimiter), nil); err != nil {
		log.WithError(err).Fatal("Failed setting configuration default")
	}

	return 1
}

func setConfigDefaults(check bool) error {
	added := 0

	// client settings
	added += setConfigDefault("clients", map[string]interface{}{
		"deluge": map[string]interface{}{
			// non struct mapped
			"enabled":       false,
			"type":          "deluge",
			"filter":        "default",
			"download_path": "/mnt/local/downloads/torrents/deluge",
			"download_path_mapping": map[string]string{
				"/downloads/torrents/deluge": "/mnt/local/downloads/torrents/deluge",
			},
			// mapped to client struct
			"host":     "localhost",
			"port":     58846,
			"login":    "localclient",
			"password": "",
			"v2":       true,
		},
	}, check)

	// filter settings
	added += setConfigDefault("filters", map[string]FilterConfiguration{
		"default": {
			Ignore: []string{
				`Label startsWith "permaseed"`,
			},
			Remove: []string{
				`Ratio > 4.0 || SeedingDays >= 15.0`,
			},
		},
	}, check)

	// were new settings added?
	if check && added > 0 {
		// unmarshal to config struct
		if err := K.Unmarshal("", &Config); err != nil {
			return errors.WithMessage(err, "failed unmarshalling configuration file")
		}

		// marshal config struct
		m, err := yaml2.Marshal(&Config)
		if err != nil {
			return errors.WithMessage(err, "failed marshalling updated configuration")
		}

		// write marshalled config to file
		err = ioutil.WriteFile(cfgPath, m, 0644)
		if err != nil {
			log.WithError(err).Error("Failed saving configuration with new options...")
			return errors.Wrap(err, "failed saving updated configuration")
		}

		// notify
		log.Info("Configuration was saved with new options!")
		log.Logger.Exit(0)
	}

	return nil
}
