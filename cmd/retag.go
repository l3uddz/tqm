package cmd

import (
	"encoding/json"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/l3uddz/tqm/client"
	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/expression"
	"github.com/l3uddz/tqm/logger"
	"github.com/l3uddz/tqm/torrentfilemap"
	"github.com/l3uddz/tqm/tracker"
)

var retagCmd = &cobra.Command{
	Use:   "retag [CLIENT]",
	Short: "Check client (only qbit) for torrents to retag",
	Long:  `This command can be used to check a torrent clients queue for torrents to retag based on its configured filters.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// init core
		if !initialized {
			initCore(true)
			initialized = true
		}

		// set log
		log := logger.GetLogger("retag")

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

		if *clientType != "qbittorrent" {
			log.Fatalf("Retagging is currently only supported for qbittorrent")
		}

		// retrieve client free space path
		clientFreeSpacePath, _ := getClientConfigString("free_space_path", clientConfig)

		// retrieve client filters
		clientFilter, err := getClientFilter(clientConfig)
		if err != nil {
			log.WithError(err).Fatal("Failed retrieving client filter")
		}

		if flagFilterName != "" {
			clientFilter, err = getFilter(flagFilterName)
			if err != nil {
				log.WithError(err).Fatal("Failed retrieving specified filter")
			}
		}

		// compile client filters
		exp, err := expression.Compile(clientFilter)
		if err != nil {
			log.WithError(err).Fatal("Failed compiling client filters")
		}

		// load client object
		c, err := client.NewClient(*clientType, clientName, exp)

		ct, ok := c.(client.TagInterface)
		if !ok {
			log.Fatalf("Retagging is currently only supported for qbittorrent")
		}

		if err != nil {
			log.WithError(err).Fatalf("Failed initializing client: %q", clientName)
		}

		log.Infof("Initialized client %q, type: %s (%d trackers)", clientName, ct.Type(), tracker.Loaded())

		// connect to client
		if err := ct.Connect(); err != nil {
			log.WithError(err).Fatal("Failed connecting")
		} else {
			log.Debugf("Connected to client")
		}

		// get free disk space (can/will be used by filters)
		if clientFreeSpacePath != nil {
			space, err := ct.GetCurrentFreeSpace(*clientFreeSpacePath)
			if err != nil {
				log.WithError(err).Warnf("Failed retrieving free-space for: %q", *clientFreeSpacePath)
			} else {
				log.Infof("Retrieved free-space for %q: %v (%.2f GB)", *clientFreeSpacePath,
					humanize.IBytes(uint64(space)), ct.GetFreeSpace())
			}
		}

		// retrieve torrents
		torrents, err := ct.GetTorrents()
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

		// Verify tags exist on client
		var tagList []string = []string{}
		for _, v := range exp.Tags {
			tagList = append(tagList, v.Name)
		}
		if err := ct.CreateTags(tagList); err != nil {
			log.WithError(err).Fatal("Failed to create tags on client")
		} else {
			log.Infof("Verified tags exist on client")
		}

		// relabel torrents that meet the filter criteria
		if err := retagEligibleTorrents(log, ct, torrents, tfm); err != nil {
			log.WithError(err).Fatal("Failed retagging eligible torrents...")
		}
	},
}

func init() {
	rootCmd.AddCommand(retagCmd)

	retagCmd.Flags().StringVar(&flagFilterName, "filter", "", "Filter to use instead of client")
}
