[![made-with-golang](https://img.shields.io/badge/Made%20with-Golang-blue.svg?style=flat-square)](https://golang.org/)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%203-blue.svg?style=flat-square)](https://github.com/l3uddz/tqm/blob/master/LICENSE.md)
[![last commit (develop)](https://img.shields.io/github/last-commit/l3uddz/tqm/develop.svg?colorB=177DC1&label=Last%20Commit&style=flat-square)](https://github.com/l3uddz/tqm/commits/develop)
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
      - Downloaded == false && !IsUnregistered()
      - SeedingHours < 26 && !IsUnregistered()
      # misc
      - FreeSpaceSet && FreeSpaceGB > 2048 && !IsUnregistered()
      # permaseed / un-sorted (unless torrent has been deleted)
      - Label in ["permaseed-mine", "permaseed-btn", "permaseed-hdb", "permaseed-ptp", "permaseed-bhd", "permaseed-nbl", "permaseed-ufc", "radarr", "sonarr", "lidarr"] && !IsUnregistered()
    remove:
      # general
      - IsUnregistered()
      # imported
      - Label in ["sonarr-imported"] && (Ratio > 4.0 || SeedingDays >= 15.0)
      # ipt
      - Label in ["autoremove-ipt"] && (Ratio > 4.0 || SeedingDays >= 15.0)
      # mtv
      - Label in ["autoremove-mtv"] && (Ratio > 4.0 || SeedingDays >= 15.0)
      # hdt
      - Label in ["autoremove-hdt"] && (Ratio > 4.0 || SeedingDays >= 15.0)
```

## Supported Clients

- [x] Deluge
- [x] qBittorrent
- [ ] rTorrent

## Example Commands

1. Manage - Retrieve torrent client queue and run against configured filters

`tqm manage deluge --dry-run`

`tqm manage deluge`

`tqm manage qbt --dry-run`

`tqm manage qbt`

2. Orphan - Retrieve torrent client queue and local files/folders in download_path, remove orphan files/folders

`tqm orphan deluge --dry-run`

`tqm orphan deluge`

`tqm orphan qbt --dry-run`

`tqm orphan qbt`

***

## Notes

`FreeSpaceSet` and `FreeSpaceGB` are currently only supported for the following clients (when `free_space_path` is set):

- [x] Deluge
- [ ] qBittorrent

`FreeSpaceGB` will only be set **once** before torrents are evaluated against the chosen filter. 

This means as torrents are removed, `FreeSpaceGB` will not change.

# Donate

If you find this project helpful, feel free to make a small donation to the developer:

  - [Monzo](https://monzo.me/today): Credit Cards, Apple Pay, Google Pay

  - [Paypal: l3uddz@gmail.com](https://www.paypal.me/l3uddz)
  
  - [GitHub Sponsor](https://github.com/sponsors/l3uddz): GitHub matches contributions for first 12 months.

  - BTC: 3CiHME1HZQsNNcDL6BArG7PbZLa8zUUgjL