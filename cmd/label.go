package cmd

import (
	"github.com/dustin/go-humanize"
	"github.com/l3uddz/tqm/client"
	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/expression"
	"github.com/l3uddz/tqm/logger"
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label [CLIENT]",
	Short: "Check torrent client for torrents to relabel",
	Long:  `This command can be used to check a torrent clients queue for torrents to relabel based on its configured filters.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// init core
		initCore(true)

		// set log
		log := logger.GetLogger("label")

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
		exp, err := expression.Compile(clientName, clientFilter)
		if err != nil {
			log.WithError(err).Fatal("Failed compiling client filters")
		}

		// load client object
		c, err := client.NewClient(*clientType, clientName, exp)
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

		log.Info("Running")
		if err := c.SetTorrentLabel("a656e6ec1d59d7262c7644f21b7676978540ff2b", "permaseed-btn"); err != nil {
			log.WithError(err).Fatal("Failed setting torrent label")
		}

		// retrieve torrents
		//torrents, err := c.GetTorrents()
		//if err != nil {
		//	log.WithError(err).Fatal("Failed retrieving torrents")
		//} else {
		//	log.Infof("Retrieved %d torrents", len(torrents))
		//}
		//
		//if flagLogLevel > 1 {
		//	if b, err := json.Marshal(torrents); err != nil {
		//		log.WithError(err).Error("Failed marshalling torrents")
		//	} else {
		//		log.Trace(string(b))
		//	}
		//}
		//
		//// create map of files associated to torrents (via hash)
		//tfm := torrentfilemap.New(torrents)
		//log.Infof("Mapped torrents to %d unique torrent files", tfm.Length())

		// relabel torrents that meet the filter criteria

	},
}

func init() {
	rootCmd.AddCommand(labelCmd)
}
