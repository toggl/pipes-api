package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type workspaceResponse struct {
	Workspace *Workspace `json:"data"`
}

func getTogglWorkspaceID(APIToken string) (int, error) {
	var workspaceID int
	url := fmt.Sprintf("%s/api/pipes/workspace", urls.TogglAPIHost[*environment])
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return workspaceID, err
	}
	req.Header.Set("User-Agent", "toggl-pipes")
	req.SetBasicAuth(APIToken, "api_token")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return workspaceID, err
	}
	var b []byte
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return workspaceID, err
	}
	if http.StatusOK != resp.StatusCode {
		return workspaceID, fmt.Errorf("GET workspace failed %d", resp.StatusCode)
	}

	var response workspaceResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return workspaceID, err
	}

	return response.Workspace.ID, nil
}

func postPipesAPI(APIToken, pipeID string, payload interface{}) ([]byte, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/pipes/%s", urls.TogglAPIHost[*environment], pipeID)
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "toggl-pipes")
	req.SetBasicAuth(APIToken, "api_token")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if 200 != resp.StatusCode {
		return b, fmt.Errorf("%s failed with status code %d", url, resp.StatusCode)
	}
	log.Println("Response from Toggl API:", string(b), "-", time.Since(start))
	return b, nil
}
