package cmd

import (
	"github.com/l3uddz/tqm/client"
	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/logger"
	"github.com/l3uddz/tqm/torrentfilemap"
	"github.com/spf13/cobra"
)

var manageCmd = &cobra.Command{
	Use:   "manage [CLIENT]",
	Short: "Check torrent client for torrents to remove",
	Long:  `This command can be used to check a torrent clients queue for torrents to remove based on its configured filters.`,

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
		clientType, err := getClientType(clientConfig)
		if err != nil {
			log.WithError(err).Fatal("Failed determining client type")
		}

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

		// retrieve torrents
		torrents, err := c.GetTorrents()
		if err != nil {
			log.WithError(err).Fatal("Failed retrieving torrents")
		} else {
			log.Infof("Retrieved %d torrents", len(torrents))
		}

		// create map of files associated to torrents (via hash)
		tfm := torrentfilemap.New(torrents)
		log.Infof("Mapped torrents to %d unique torrent files", tfm.Length())

		// remove torrents that should be ignored
		if err := removeIgnoredTorrents(log, c, torrents); err != nil {
			log.WithError(err).Fatal("Failed removing torrents that should be ignored...")
		}

		// remove torrents that should be removed
		if err := removeEligibleTorrents(log, c, torrents, tfm); err != nil {
			log.WithError(err).Fatal("Failed removing eligible torrents...")
		}
	},
}

func init() {
	rootCmd.AddCommand(manageCmd)
}
