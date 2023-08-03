package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Video struct {
	UID string `json:"uid"`
}

type VideoList struct {
	Result []Video `json:"result"`
}

type VideoDownloadResp struct {
	Success bool `json:"success"`
	Result  struct {
		Default struct {
			Status          string  `json:"status"`
			URL             string  `json:"url"`
			PercentComplete float64 `json:"percentComplete"`
		} `json:"default"`
	} `json:"result"`
}

type Download struct {
	VideoID string
	URL     string
}

func main() {
	accountID := os.Getenv("CF_ACCOUNT_ID")
	authKey := "Bearer " + os.Getenv("CF_AUTH_KEY")

	// Get the list of videos
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/stream", accountID), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", authKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Parse the response
	var videoList VideoList
	err = json.NewDecoder(resp.Body).Decode(&videoList)
	if err != nil {
		panic(err)
	}

	isProd := os.Getenv("CF_PROD")

	fmt.Printf("Fetched %d videos...\n", len(videoList.Result))

	downloadLinks := make([]Download, 0)

	// Trigger download for each video
	for _, video := range videoList.Result {
		if isProd != "" {
			req, err := http.NewRequest("POST",
				fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/stream/%s/downloads", accountID, video.UID), nil)
			if err != nil {
				fmt.Println("Error downloading video:", video.UID, err)
				continue
			}

			req.Header.Add("Authorization", authKey)
			req.Header.Add("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}

			defer resp.Body.Close()

			var vidDownResp VideoDownloadResp
			err = json.NewDecoder(resp.Body).Decode(&vidDownResp)
			if err != nil {
				panic(err)
			}
			fmt.Println("Video download triggered:", video.UID)
			fmt.Println("Download trigger response:", vidDownResp)
			if vidDownResp.Success {
				downloadLinks = append(downloadLinks, Download{
					VideoID: video.UID,
					URL:     vidDownResp.Result.Default.URL,
				})
			}
		} else {
			fmt.Printf("Video ID: %s\n", video.UID)
		}
	}

	fmt.Println("Waiting for videos to be ready for download...")
	time.Sleep(15 * time.Second)
	// TODO: Insert logic to check if the last video ID's percent complete is 100

	// Download the video
	for _, downloadLink := range downloadLinks {
		fmt.Println("Downloading video: ", downloadLink.VideoID)

		resp, err := http.Get(downloadLink.URL)
		if err != nil {
			fmt.Println("Error downloading video:", downloadLink, err)
			continue
		}

		defer resp.Body.Close()

		// Create the file
		out, err := os.Create(fmt.Sprintf("%s.mp4", downloadLink.VideoID))
		if err != nil {
			fmt.Println("Error creating file:", err)
			continue
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			fmt.Println("Error writing to file:", err)
		}
	}
}
