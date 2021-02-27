package cmd

import (
	"github.com/dustin/go-humanize"
	"github.com/l3uddz/tqm/client"
	"github.com/l3uddz/tqm/config"
	"github.com/l3uddz/tqm/torrentfilemap"
	"github.com/sirupsen/logrus"
	"time"
)

// relabel torrent that meet required filters
func relabelEligibleTorrents(log *logrus.Entry, c client.Interface, torrents map[string]config.Torrent,
	tfm *torrentfilemap.TorrentFileMap) error {
	// vars
	ignoredTorrents := 0
	nonUniqueTorrents := 0
	relabeledTorrents := 0

	// iterate torrents
	for h, t := range torrents {
		if !tfm.IsUnique(t) {
			// torrent file is not unique, files are contained within another torrent
			// so we cannot safely change the label in-case of auto move
			nonUniqueTorrents++
			log.Warnf("Skipping non unique torrent: %v: %+v", h, t)
			continue
		}

		// should we relabel torrent?
		label, relabel, err := c.ShouldRelabel(&t)
		if err != nil {
			// error while determining whether to relabel torrent
			log.WithError(err).Errorf("Failed determining whether to relabel %q: %+v", t.Name, t)
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
	log.Infof("Relabeled torrents: %d", relabeledTorrents)
	return nil
}

// remove torrents that meet remove filters
func removeEligibleTorrents(log *logrus.Entry, c client.Interface, torrents map[string]config.Torrent,
	tfm *torrentfilemap.TorrentFileMap) error {
	// vars
	ignoredTorrents := 0
	softRemoveTorrents := 0
	hardRemoveTorrents := 0
	errorRemoveTorrents := 0
	var removedTorrentBytes int64 = 0

	// iterate torrents
	for h, t := range torrents {
		// should we ignore this torrent?
		ignore, err := c.ShouldIgnore(&t)
		if err != nil {
			// error while determining whether to ignore torrent
			log.WithError(err).Errorf("Failed determining whether to ignore %q: %+v", t.Name, t)
			delete(torrents, h)
			continue
		} else if ignore {
			// torrent met ignore filter
			log.Tracef("Ignoring torrent %s: %s", h, t.Name)
			delete(torrents, h)
			ignoredTorrents++
			continue
		}

		// should we remove this torrent?
		remove, err := c.ShouldRemove(&t)
		if err != nil {
			log.WithError(err).Errorf("Failed determining whether to remove %q: %+v", t.Name, t)
			// dont do any further operations on this torrent, but keep in the torrent file map
			delete(torrents, h)
			continue
		} else if !remove {
			// torrent did not meet the remove filters
			log.Tracef("Not removing %s: %s", h, t.Name)
			continue
		}

		// torrent meets the remove filters
		// are the files unique and eligible for a hard deletion (remove data)
		uniqueTorrent := tfm.IsUnique(t)
		removeMode := "Soft"

		if uniqueTorrent {
			// this torrent contains files found within other torrents (dont remove its data)
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

	// show result
	log.Info("-----")
	log.Infof("Ignored torrents: %d", ignoredTorrents)
	log.WithField("reclaimed_space", humanize.IBytes(uint64(removedTorrentBytes))).
		Infof("Removed torrents: %d hard, %d soft and %d failures",
			hardRemoveTorrents, softRemoveTorrents, errorRemoveTorrents)
	return nil
}
