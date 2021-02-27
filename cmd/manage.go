package cmd

import (
	"encoding/json"
	"github.com/dustin/go-humanize"
	"github.com/l3uddz/tqm/client"
	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/logger"
	"github.com/l3uddz/tqm/torrentfilemap"
	"github.com/spf13/cobra"
)

var (
	flagLabel bool
)

var manageCmd = &cobra.Command{
	Use:   "manage [CLIENT]",
	Short: "Check torrent client for torrents to remove/relabel",
	Long:  `This command can be used to check a torrent clients queue for torrents to remove/relabel based on its configured filters.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// init core
		initCore(true)

		// set log
		log := logger.GetLogger("manage")

		// retrieve client object
		clientName := args[0]
		clientConfig, ok := config.Config.Clients[clientName]
		if !ok {
			log.Fatalf("No client configuration found for: %q", clientName)
		}

		// validate client is enabled
		if err := validateClientEnabled(clientConfig); err != nil {
			log.WithError(err).Fatal("Failed validating client is enabled")
		}

		// retrieve client type
		clientType, err := getClientConfigString("type", clientConfig)
		if err != nil {
			log.WithError(err).Fatal("Failed determining client type")
		}

		// retrieve client free space path
		clientFreeSpacePath, _ := getClientConfigString("free_space_path", clientConfig)

		// retrieve client filters
		clientFilter, err := getClientFilter(clientConfig)
		if err != nil {
			log.WithError(err).Fatal("Failed retrieving client filter")
		}

		// compile client filters
		ignoreExpressions, removeExpressions, err := compileExpressions(clientName, clientFilter)
		if err != nil {
			log.WithError(err).Fatal("Failed compiling client filters")
		}

		// load client object
		c, err := client.NewClient(*clientType, clientName, ignoreExpressions, removeExpressions)
		if err != nil {
			log.WithError(err).Fatalf("Failed initializing client: %q", clientName)
		}

		log.Infof("Initialized client %q, type: %s", clientName, c.Type())

		// connect to client
		if err := c.Connect(); err != nil {
			log.WithError(err).Fatal("Failed connecting")
		} else {
			log.Debugf("Connected to client")
		}

		// get free disk space (can/will be used by filters)
		if clientFreeSpacePath != nil {
			space, err := c.GetCurrentFreeSpace(*clientFreeSpacePath)
			if err != nil {
				log.WithError(err).Warnf("Failed retrieving free-space for: %q", *clientFreeSpacePath)
			} else {
				log.Infof("Retrieved free-space for %q: %v (%.2f GB)", *clientFreeSpacePath,
					humanize.IBytes(uint64(space)), c.GetFreeSpace())
			}
		}

		// retrieve torrents
		torrents, err := c.GetTorrents()
		if err != nil {
			log.WithError(err).Fatal("Failed retrieving torrents")
		} else {
			log.Infof("Retrieved %d torrents", len(torrents))
		}

		if flagLogLevel > 1 {
			if b, err := json.Marshal(torrents); err != nil {
				log.WithError(err).Error("Failed marshalling torrents")
			} else {
				log.Trace(string(b))
			}
		}

		// create map of files associated to torrents (via hash)
		tfm := torrentfilemap.New(torrents)
		log.Infof("Mapped torrents to %d unique torrent files", tfm.Length())

		// remove torrents that are not ignored and match remove criteria
		if err := removeEligibleTorrents(log, c, torrents, tfm); err != nil {
			log.WithError(err).Fatal("Failed removing eligible torrents...")
		}

		// relabel torrents
		if flagLabel {
			log.Info("-----")
			if err := labelCmd.Execute(); err != nil {
				log.WithError(err).Fatal("Failed executing label command")
			}
		}
	},
}

func init() {
	manageCmd.Flags().BoolVar(&flagLabel, "label", false, "Relabel torrents")

	rootCmd.AddCommand(manageCmd)
}
