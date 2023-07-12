package cmd

import (
	"sort"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"

	"github.com/l3uddz/tqm/client"
	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/torrentfilemap"
)

// relabel torrent that meet required filters
func relabelEligibleTorrents(log *logrus.Entry, c client.Interface, torrents map[string]config.Torrent,
	tfm *torrentfilemap.TorrentFileMap) error {
	// vars
	ignoredTorrents := 0
	nonUniqueTorrents := 0
	relabeledTorrents := 0
	errorRelabelTorrents := 0

	// iterate torrents
	for h, t := range torrents {
		if !tfm.IsUnique(t) {
			// torrent file is not unique, files are contained within another torrent
			// so we cannot safely change the label in-case of auto move
			nonUniqueTorrents++
			log.Warnf("Skipping non unique torrent: %+v", t)
			continue
		}

		// should we relabel torrent?
		label, relabel, err := c.ShouldRelabel(&t)
		if err != nil {
			// error while determining whether to relabel torrent
			log.WithError(err).Errorf("Failed determining whether to relabel: %+v", t)
			continue
		} else if !relabel {
			// torrent did not meet the relabel filters
			log.Tracef("Not relabeling %s: %s", h, t.Name)
			ignoredTorrents++
			continue
		}

		// relabel
		log.Info("-----")
		log.Infof("Relabeling: %q - %s", t.Name, label)
		log.Infof("Ratio: %.3f / Seed days: %.3f / Seeds: %d / Label: %s / Tracker: %s / "+
			"Tracker Status: %q", t.Ratio, t.SeedingDays, t.Seeds, t.Label, t.TrackerName, t.TrackerStatus)

		if !flagDryRun {
			if err := c.SetTorrentLabel(t.Hash, label); err != nil {
				log.WithError(err).Fatalf("Failed relabeling torrent: %+v", t)
				errorRelabelTorrents++
				continue
			}

			log.Info("Relabeled")
			time.Sleep(5 * time.Second)
		} else {
			log.Warn("Dry-run enabled, skipping relabel...")
		}

		relabeledTorrents++
	}

	// show result
	log.Info("-----")
	log.Infof("Ignored torrents: %d", ignoredTorrents)
	if nonUniqueTorrents > 0 {
		log.Infof("Non-unique torrents: %d", nonUniqueTorrents)
	}
	log.Infof("Relabeled torrents: %d, %d failures", relabeledTorrents, errorRelabelTorrents)
	return nil
}

type HashTorrentPair struct {
	Hash    string
	Torrent config.Torrent
}

func getHashTorrentPairs(torrents map[string]config.Torrent) []HashTorrentPair {
	hashTorrentPairs := make([]HashTorrentPair, 0, len(torrents))
	for h, t := range torrents {
		hashTorrentPairs = append(hashTorrentPairs, HashTorrentPair{Hash: h, Torrent: t})
	}
	return hashTorrentPairs
}

func sortTorrentsByAddedSeconds(torrents []HashTorrentPair) {
	sort.Slice(torrents, func(i, j int) bool {
		return torrents[i].Torrent.AddedSeconds < torrents[j].Torrent.AddedSeconds
	})
}

func filterTorrents(hashTorrentPairs []HashTorrentPair, log *logrus.Entry, c client.Interface, torrents map[string]config.Torrent) ([]HashTorrentPair, int, int) {
	removeTorrents := make([]HashTorrentPair, 0)
	ignoredTorrents := 0
	errorRemoveTorrents := 0

	for _, pair := range hashTorrentPairs {
		h := pair.Hash
		t := pair.Torrent

		// should we ignore this torrent?
		ignore, err := c.ShouldIgnore(&t)
		if err != nil {
			log.WithError(err).Errorf("Failed determining whether to ignore: %+v", t)
			delete(torrents, h)
			continue
		} else if ignore {
			log.Tracef("Ignoring torrent %s: %s", h, t.Name)
			delete(torrents, h)
			ignoredTorrents++
			continue
		}

		// should we remove this torrent?
		remove, err := c.ShouldRemove(&t)
		if err != nil {
			log.WithError(err).Errorf("Failed determining whether to remove: %+v", t)
			delete(torrents, h)
			continue
		} else if remove {
			removeTorrents = append(removeTorrents, pair)
		}
	}

	return removeTorrents, ignoredTorrents, errorRemoveTorrents
}

func processRemoveTorrents(removeCount int, removeTorrents []HashTorrentPair, log *logrus.Entry, c client.Interface, tfm *torrentfilemap.TorrentFileMap, torrents map[string]config.Torrent) (int64, int, int, int) {
	softRemoveTorrents := 0
	hardRemoveTorrents := 0
	errorRemoveTorrents := 0
	var removedTorrentBytes int64 = 0

	for i := 0; i < removeCount; i++ {
		pair := removeTorrents[i]

		h := pair.Hash
		t := pair.Torrent

		// torrent meets the remove filters
		// are the files unique and eligible for a hard deletion (remove data)
		uniqueTorrent := tfm.IsUnique(t)
		removeMode := "Soft"

		if uniqueTorrent {
			// this torrent does not contains files found within other torrents (remove its data)
			removeMode = "Hard"
		}

		// remove the torrent
		log.Info("-----")
		if !t.FreeSpaceSet {
			log.Infof("%s removing: %q - %s", removeMode, t.Name, humanize.IBytes(uint64(t.DownloadedBytes)))
		} else {
			// show current free-space as well
			log.Infof("%s removing: %q - %s - %.2f GB", removeMode, t.Name,
				humanize.IBytes(uint64(t.DownloadedBytes)), t.FreeSpaceGB())
		}

		log.Infof("Ratio: %.3f / Seed days: %.3f / Seeds: %d / Label: %s / Tracker: %s / "+
			"Tracker Status: %q", t.Ratio, t.SeedingDays, t.Seeds, t.Label, t.TrackerName, t.TrackerStatus)

		if !flagDryRun {
			// do remove
			removed, err := c.RemoveTorrent(t.Hash, uniqueTorrent)
			if err != nil {
				log.WithError(err).Fatalf("Failed removing torrent: %+v", t)
				// dont remove from torrents file map, but prevent further operations on this torrent
				delete(torrents, h)
				errorRemoveTorrents++
				continue
			} else if !removed {
				log.Error("Failed removing torrent...")
				// dont remove from torrents file map, but prevent further operations on this torrent
				delete(torrents, h)
				errorRemoveTorrents++
				continue
			} else {
				log.Info("Removed")

				// increase free space (if its a hard remove)
				if uniqueTorrent && t.FreeSpaceSet {
					log.Tracef("Increasing free space by: %s", humanize.IBytes(uint64(t.DownloadedBytes)))
					c.AddFreeSpace(t.DownloadedBytes)
					log.Tracef("New free space: %.2f GB", c.GetFreeSpace())
				}

				time.Sleep(1 * time.Second)
			}
		} else {
			log.Warn("Dry-run enabled, skipping remove...")
		}

		if uniqueTorrent {
			// increased hard removed counters
			removedTorrentBytes += t.DownloadedBytes
			hardRemoveTorrents++
		} else {
			// increase soft remove counters
			softRemoveTorrents++
		}

		// remove the torrent from the torrent file map
		tfm.Remove(t)
		delete(torrents, h)
	}

	return removedTorrentBytes, hardRemoveTorrents, softRemoveTorrents, errorRemoveTorrents
}

func removeEligibleTorrents(log *logrus.Entry, c client.Interface, torrents map[string]config.Torrent,
	tfm *torrentfilemap.TorrentFileMap) error {

	torrentRetentionLimit := config.Config.TorrentRetentionLimit

	hashTorrentPairs := getHashTorrentPairs(torrents)
	sortTorrentsByAddedSeconds(hashTorrentPairs)

	//Filter out all torrents that can be removed
	removeTorrents, ignoredTorrents, errorRemoveTorrents := filterTorrents(hashTorrentPairs, log, c, torrents)

	// Sort the removeTorrents slice by AddedSeconds to make sure oldest torrents are removed first
	sortTorrentsByAddedSeconds(removeTorrents)

	NonRemovableTorrent := len(hashTorrentPairs) - len(removeTorrents)
	removeCount := torrentRetentionLimit - NonRemovableTorrent

	if removeCount < 0 {
		removeCount = len(removeTorrents)
	} else {
		removeCount = len(removeTorrents) - removeCount
	}

	removedTorrentBytes, hardRemoveTorrents, softRemoveTorrents, errorRemoveTorrents :=
		processRemoveTorrents(removeCount, removeTorrents, log, c, tfm, torrents)

	// show result
	log.Info("-----")
	log.Infof("Ignored torrents: %d", ignoredTorrents)
	log.WithField("reclaimed_space", humanize.IBytes(uint64(removedTorrentBytes))).
		Infof("Removed torrents: %d hard, %d soft and %d failures",
			hardRemoveTorrents, softRemoveTorrents, errorRemoveTorrents)
	return nil
}
