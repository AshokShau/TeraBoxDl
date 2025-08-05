package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type TeraResponse struct {
	Errno      int64  `json:"errno"`
	RequestID  int64  `json:"request_id"`
	ServerTime int64  `json:"server_time"`
	ShareID    string `json:"share_id"`
	UK         string `json:"uk"`
	List       []struct {
		Category       int64  `json:"category"`
		FsID           int64  `json:"fs_id"`
		IsDir          int    `json:"isdir"`
		ServerFilename string `json:"server_filename"`
		Size           int64  `json:"size"`
		Dlink          string `json:"dlink"`
		DirectLink     string `json:"direct_link"`
		StreamURL      string `json:"stream_url"`
		Thumbs         struct {
			URL1 string `json:"url1"`
			URL2 string `json:"url2"`
			URL3 string `json:"url3"`
			Icon string `json:"icon"`
		} `json:"thumbs"`
	} `json:"list"`
}

func getTeraBoxInfo(shareURL string) (*TeraResponse, error) {
	apiURL := "https://info.fallenapi.fun/tera?url=" + shareURL

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf(`{"error":"Invalid request body"}`)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(`{"error":"Invalid request body"}`)
	}

	var result TeraResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.New(string(body))
	}

	return &result, nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
