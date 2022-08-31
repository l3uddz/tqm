[![made-with-golang](https://img.shields.io/badge/Made%20with-Golang-blue.svg?style=flat-square)](https://golang.org/)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%203-blue.svg?style=flat-square)](https://github.com/l3uddz/tqm/blob/master/LICENSE.md)
[![Discord](https://img.shields.io/discord/381077432285003776.svg?colorB=177DC1&label=Discord&style=flat-square)](https://discord.io/cloudbox)
[![Contributing](https://img.shields.io/badge/Contributing-gray.svg?style=flat-square)](CONTRIBUTING.md)
[![Donate](https://img.shields.io/badge/Donate-gray.svg?style=flat-square)](#donate)

# tqm

CLI tool to manage your torrent client queues. Primary focus is on removing torrents that meet specific criteria.

## Example Configuration

```yaml
clients:
  deluge:
    enabled: true
    filter: default
    download_path: /mnt/local/downloads/torrents/deluge
    free_space_path: /mnt/local/downloads/torrents/deluge
    download_path_mapping:
      /downloads/torrents/deluge: /mnt/local/downloads/torrents/deluge
    host: localhost
    login: localclient
    password: password-from-/opt/deluge/auth
    port: 58846
    type: deluge
    v2: true
  qbt:
    download_path: /mnt/local/downloads/torrents/qbittorrent/completed
    download_path_mapping:
      /downloads/torrents/qbittorrent/completed: /mnt/local/downloads/torrents/qbittorrent/completed
    enabled: true
    filter: default
    type: qbittorrent
    url: https://qbittorrent.domain.com/
    user: user
    password: password
filters:
  default:
    ignore:
      # general
      - TrackerStatus contains "Tracker is down"
      - Downloaded == false && !IsUnregistered()
      - SeedingHours < 26 && !IsUnregistered()
      # permaseed / un-sorted (unless torrent has been deleted)
      - Label startsWith "permaseed-" && !IsUnregistered()
      # Filter based on qbittorrent tags (only qbit at the moment)
      - '"permaseed" in Tags && !IsUnregistered()'
    remove:
      # general
      - IsUnregistered()
      # imported
      - Label in ["sonarr-imported", "radarr-imported", "lidarr-imported"] && (Ratio > 4.0 || SeedingDays >= 15.0)
      # ipt
      - Label in ["autoremove-ipt"] && (Ratio > 3.0 || SeedingDays >= 15.0)
      # hdt
      - Label in ["autoremove-hdt"] && (Ratio > 3.0 || SeedingDays >= 15.0)
      # bhd
      - Label in ["autoremove-bhd"] && (Ratio > 3.0 || SeedingDays >= 15.0)
      # ptp
      - Label in ["autoremove-ptp"] && (Ratio > 3.0 || SeedingDays >= 15.0)
      # btn
      - Label in ["autoremove-btn"] && (Ratio > 3.0 || SeedingDays >= 15.0)
      # hdb
      - Label in ["autoremove-hdb"] && (Ratio > 3.0 || SeedingDays >= 15.0)
      # Qbit tag utilities
      - HasAllTags("480p", "bad-encode") # match if all tags are present
      - HasAnyTag("remove-me", "gross") # match if at least 1 tag is present
    label:
      # btn 1080p season packs to permaseed (all must evaluate to true)
      - name: permaseed-btn
        update:
          - Label == "sonarr-imported"
          - TrackerName == "landof.tv"
          - Name contains "1080p"
          - len(Files) >= 3

      # cleanup btn season packs to autoremove-btn (all must evaluate to true)
      - name: autoremove-btn
        update:
          - Label == "sonarr-imported"
          - TrackerName == "landof.tv"
          - not (Name contains "1080p")
          - len(Files) >= 3
    # Change qbit tags based on filters
    tag:
      - name: low-seed
      # This must be set
      # "mode: full" means tag will be added to
      # torrent if matched and removed from torrent if not
      # use `add` or `remove` to only add/remove respectivly
      # NOTE: Mode does not change the way torrents are flagged,
      # meaning, even with "mode: remove",
      # tags will be removed if the torrent does NOT match the conditions.
      # "mode: remove" simply means that tags will not be added
      # to torrents that do match.
        mode: full
        update:
          - Seeds <= 3

```
## Optional - Tracker Configuration
```yaml
trackers:
  bhd:
    api_key: your-api-key
  ptp:
    api_user: your-api-user
    api_key: your-api-key
```
Allows tqm to validate if a torrent was removed from the tracker using the tracker's own API.

Currently implements:
- Beyond-HD
- PTP


## Supported Clients

- Deluge
- qBittorrent

## Example Commands

1. Clean - Retrieve torrent client queue and remove torrents matching its configured filters

`tqm clean qbt --dry-run`

`tqm clean qbt`

2. Relabel - Retrieve torrent client queue and relabel torrents matching its configured filters

`tqm relabel qbt --dry-run`

`tqm relabel qbt`

3. Retag - Retrieve torrent client queue and retag torrents matching its configured filters

`tqm retag qbt --dry-run`

`tqm retag qbt`

4. Orphan - Retrieve torrent client queue and local files/folders in download_path, remove orphan files/folders

`tqm orphan qbt --dry-run`

`tqm orphan qbt`

***

## Notes

`FreeSpaceSet` and `FreeSpaceGB()` are currently only supported for the following clients (when `free_space_path` is set):

- Deluge
- qBittorrent

`FreeSpaceGB()` will only increase as torrents are hard-removed.

This only works with one disk referenced by `free_space_path` and will not account for torrents being on **different disks**.

# Donate

If you find this project helpful, feel free to make a small donation to the developer:

  - [Monzo](https://monzo.me/today): Credit Cards, Apple Pay, Google Pay

  - [Paypal: l3uddz@gmail.com](https://www.paypal.me/l3uddz)
  
  - [GitHub Sponsor](https://github.com/sponsors/l3uddz): GitHub matches contributions for first 12 months.

  - BTC: 3CiHME1HZQsNNcDL6BArG7PbZLa8zUUgjL
