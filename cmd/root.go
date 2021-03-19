package cmd

import (
	"fmt"
	"github.com/l3uddz/tqm/runtime"
	"github.com/l3uddz/tqm/stringutils"
	"os"
	"path/filepath"

	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/logger"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	flagLogLevel     = 0
	flagConfigFile   = "config.yaml"
	flagConfigFolder = config.GetDefaultConfigDirectory("tqm", flagConfigFile)
	flagLogFile      = "activity.log"

	flagDryRun bool

	// Global vars
	log         *logrus.Entry
	initialized bool
)

var rootCmd = &cobra.Command{
	Use:   "tqm",
	Short: "A CLI torrent queue manager",
	Long: `A CLI application that can be used to manage your torrent clients.
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Parse persistent flags
	rootCmd.PersistentFlags().StringVar(&flagConfigFolder, "config-dir", flagConfigFolder, "Config folder")
	rootCmd.PersistentFlags().StringVarP(&flagConfigFile, "config", "c", flagConfigFile, "Config file")
	rootCmd.PersistentFlags().StringVarP(&flagLogFile, "log", "l", flagLogFile, "Log file")
	rootCmd.PersistentFlags().CountVarP(&flagLogLevel, "verbose", "v", "Verbose level")

	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Dry run mode")
}

func initCore(showAppInfo bool) {
	// Set core variables
	if !rootCmd.PersistentFlags().Changed("config") {
		flagConfigFile = filepath.Join(flagConfigFolder, flagConfigFile)
	}
	if !rootCmd.PersistentFlags().Changed("log") {
		flagLogFile = filepath.Join(flagConfigFolder, flagLogFile)
	}

	// Init Logging
	if err := logger.Init(flagLogLevel, flagLogFile); err != nil {
		log.WithError(err).Fatal("Failed to initialize logging")
	}

	log = logger.GetLogger("app")

	// Init Config
	if err := config.Init(flagConfigFile); err != nil {
		log.WithError(err).Fatal("Failed to initialize config")
	}

	// Show App Info
	if showAppInfo {
		showUsing()
	}
}

func showUsing() {
	// show app info
	log.Infof("Using %s = %s (%s@%s)", stringutils.LeftJust("VERSION", " ", 10),
		runtime.Version, runtime.GitCommit, runtime.Timestamp)
	logger.ShowUsing()
	config.ShowUsing()
	log.Info("------------------")
}

func validateClientEnabled(clientConfig map[string]interface{}) error {
	v, ok := clientConfig["enabled"]
	if !ok {
		return fmt.Errorf("no enabled setting found in client configuration: %+v", clientConfig)
	} else {
		enabled, ok := v.(bool)
		if !ok || !enabled {
			return errors.New("client is not enabled")
		}
	}

	return nil
}

func getClientConfigString(setting string, clientConfig map[string]interface{}) (*string, error) {
	v, ok := clientConfig[setting]
	if !ok {
		return nil, fmt.Errorf("no %q setting found in client configuration: %+v", setting, clientConfig)
	}

	value, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("failed type-asserting %q of client: %#v", setting, v)
	}

	return &value, nil
}

func getClientDownloadPathMapping(clientConfig map[string]interface{}) (map[string]string, error) {
	v, ok := clientConfig["download_path_mapping"]
	if !ok {
		return nil, nil
	}

	tmp, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed type-asserting download_path_mapping of client: %#v", v)
	}

	clientDownloadPathMapping := make(map[string]string)
	for k, v := range tmp {
		if vv, ok := v.(string); ok {
			clientDownloadPathMapping[k] = vv
		} else {
			return nil, fmt.Errorf("failed type-asserting download_path_mapping of client for %q: %#v", k, v)
		}
	}

	return clientDownloadPathMapping, nil
}

func getClientFilter(clientConfig map[string]interface{}) (*config.FilterConfiguration, error) {
	v, ok := clientConfig["filter"]
	if !ok {
		return nil, fmt.Errorf("no filter setting found in client configuration: %+v", clientConfig)
	}

	clientFilterName, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("failed type-asserting filter of client: %#v", v)
	}

	clientFilter, ok := config.Config.Filters[clientFilterName]
	if !ok {
		return nil, fmt.Errorf("failed finding configuration of filter: %+v", clientFilterName)
	}

	return &clientFilter, nil
}

func getFilter(filterName string) (*config.FilterConfiguration, error) {
	clientFilter, ok := config.Config.Filters[filterName]
	if !ok {
		return nil, fmt.Errorf("failed finding configuration of filter: %+v", filterName)
	}

	return &clientFilter, nil
}
