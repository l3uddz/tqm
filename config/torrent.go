package config

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	unregisteredStatuses = []string{
		"not registered with this tracker",
		"torrent is not authorized for use on this tracker",
		"torrent is not found",
		"torrent not found",
		"torrent has been nuked",
		"torrent does not exist",
		"unregistered torrent",
	}
)

type Torrent struct {
	// torrent
	Hash            string   `json:"Hash"`
	Name            string   `json:"Name"`
	Path            string   `json:"Path"`
	TotalBytes      int64    `json:"TotalBytes"`
	DownloadedBytes int64    `json:"DownloadedBytes"`
	State           string   `json:"State"`
	Files           []string `json:"Files"`
	Downloaded      bool     `json:"Downloaded"`
	Seeding         bool     `json:"Seeding"`
	Ratio           float32  `json:"Ratio"`
	AddedSeconds    int64    `json:"AddedSeconds"`
	AddedHours      float32  `json:"AddedHours"`
	AddedDays       float32  `json:"AddedDays"`
	SeedingSeconds  int64    `json:"SeedingSeconds"`
	SeedingHours    float32  `json:"SeedingHours"`
	SeedingDays     float32  `json:"SeedingDays"`
	Label           string   `json:"Label"`
	Seeds           int64    `json:"Seeds"`
	Peers           int64    `json:"Peers"`

	// set by client on GetCurrentFreeSpace
	FreeSpaceGB  func() float64 `json:"-"`
	FreeSpaceSet bool           `json:"-"`

	// tracker
	TrackerName   string `json:"TrackerName"`
	TrackerStatus string `json:"TrackerStatus"`
}

func (t *Torrent) IsUnregistered() bool {
	if t.TrackerStatus == "" {
		return false
	}

	status := strings.ToLower(t.TrackerStatus)
	for _, v := range unregisteredStatuses {
		// unregistered tracker status found?
		if strings.Contains(status, v) {
			return true
		}
	}
	if strings.Contains(t.TrackerName, "tracker.beyond-hd.me") {
		return IsUnregisteredAPI(t)
	}

	return false
}

type BHDResponse struct {
	StatusCode int `json:"status_code"`
	Page       int `json:"page"`
	Results    []struct {
		ID             int     `json:"id"`
		Name           string  `json:"name"`
		FolderName     string  `json:"folder_name"`
		InfoHash       string  `json:"info_hash"`
		Size           int64   `json:"size"`
		Category       string  `json:"category "`
		Type           string  `json:"type "`
		Seeders        int     `json:"seeders "`
		Leechers       int     `json:"leechers "`
		TimesCompleted int     `json:"times_completed "`
		ImdbID         string  `json:"imdb_id "`
		TmdbID         string  `json:"tmdb_id "`
		BhdRating      int     `json:"bhd_rating "`
		TmdbRating     float64 `json:"tmdb_rating "`
		ImdbRating     float64 `json:"imdb_rating "`
		TvPack         int     `json:"tv_pack "`
		Promo25        int     `json:"promo25 "`
		Promo50        int     `json:"promo50 "`
		Promo75        int     `json:"promo75 "`
		Freeleech      int     `json:"freeleech "`
		Rewind         int     `json:"rewind "`
		Refund         int     `json:"refund "`
		Limited        int     `json:"limited "`
		Rescue         int     `json:"rescue "`
		BumpedAt       string  `json:"bumped_at "`
		CreatedAt      string  `json:"created_at "`
		URL            string  `json:"url "`
	} `json:"results"`
	TotalPages   int  `json:"total_pages"`
	TotalResults int  `json:"total_results"`
	Success      bool `json:"success"`
}

func IsUnregisteredAPI(t *Torrent) bool {
	if strings.Contains(t.TrackerName, "tracker.beyond-hd.me") {
		apikey := Config.Trackers["BHD"].APIKey
		httpposturl := "https://beyond-hd.me/api/torrents/" + apikey

		var jsonData = []byte(`{
		"info_hash": "` + t.Hash + `",
		"action": "search"
		}`)
		request, err := http.NewRequest("POST", httpposturl, bytes.NewBuffer(jsonData))
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			log.WithError(err).Errorf("Can not contact Beyond HD API: %+v", t)
			return false
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(response.Body)

		body, _ := ioutil.ReadAll(response.Body)

		var resultParse BHDResponse
		if err := json.Unmarshal(body, &resultParse); err != nil { // Parse []byte to go struct pointer
			log.WithError(err).Errorf("Can not unmarshal API JSON response")
			return false
		}

		if resultParse.TotalResults < 1 {
			return true
		}
	}
	return false
}
