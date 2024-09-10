package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
)

const (
	apiURL    = "https://api.twitch.tv/helix"
	idURL     = "https://id.twitch.tv/oauth2/token"
	apiKey    = "cotxsalhlctv8z572f7fant4b0sc3u"
	apiSecret = "gaofxvult280l3sbz8n6btvk5fdswp"
)

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

var (
	tokenMutex sync.Mutex
	tokenOnce  sync.Once
)

func getNewApiToken() (string, error) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	data := url.Values{}
	data.Set("client_id", apiKey)
	data.Set("client_secret", apiSecret)
	data.Set("grant_type", "client_credentials")

	resp, err := http.PostForm(idURL, data)
	if err != nil {
		return "", fmt.Errorf("error making API request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return "", fmt.Errorf("API request not valid, status code: %d, response: %s", resp.StatusCode, bodyString)
	}

	var authResponse AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	apiFile := homeDir + "/.cache/twitch/api"
	if err := os.WriteFile(apiFile, []byte(authResponse.AccessToken), 0644); err != nil {
		return "", fmt.Errorf("error writing API token to file: %v", err)
	}

	return authResponse.AccessToken, nil
}

func getApiToken() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	apiFile := homeDir + "/.cache/twitch/api"
	content, err := os.ReadFile(apiFile)
	if err != nil {
		newToken, err := getNewApiToken()
		if err != nil {
			return "", err
		}
		content = []byte(newToken)
	}

	return string(content), nil
}

func sendRequest(endpoint string, accessToken string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Client-ID", apiKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Check if the token is invalid
	if resp.StatusCode == http.StatusUnauthorized {
		// Get a new token
		var newToken string
		var err error
		tokenOnce.Do(func() {
			newToken, err = getNewApiToken()
		})
		if err != nil {
			return nil, err
		}

		// Retry the request with the new token
		req.Header.Set("Authorization", "Bearer "+newToken)
		resp, err = client.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, err
}

func GetStreamData(endpoint string) ([]map[string]interface{}, error) {
	accessToken, err := getApiToken()
	if err != nil {
		return nil, err
	}

	resp, err := sendRequest(endpoint, accessToken)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d: %v", resp.StatusCode, err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	streamData := make([]map[string]interface{}, len(data))
	for i, item := range data {
		streamData[i] = item.(map[string]interface{})
	}

	return streamData, nil
}
