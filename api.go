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
	apiURL = "https://api.twitch.tv/helix"
	idURL  = "https://id.twitch.tv/oauth2/token"
	// I'm tired of keeping keys private in git and nix. It's such a hassle. Just don't get the keys banned.
	apiKey    = "cotxsalhlctv8z572f7fant4b0sc3u"
	apiSecret = "gaofxvult280l3sbz8n6btvk5fdswp"
)

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

func getNewApiToken() string {
	data := url.Values{}
	data.Set("client_id", apiKey)
	data.Set("client_secret", apiSecret)
	data.Set("grant_type", "client_credentials")

	resp, err := http.PostForm(idURL, data)
	if err != nil {
		fmt.Println("Error making API request:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Printf("API request not valid, status code: %d, response: %s\n", resp.StatusCode, bodyString)
		os.Exit(1)
	}

	var authResponse AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		fmt.Println("Error decoding response:", err)
		os.Exit(1)
	}

	return authResponse.AccessToken
}

var tokenMutex sync.Mutex

func writeNewToken() {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Failed to get user home directory:", err)
		os.Exit(1)
	}

	apiFile := homeDir + "/.cache/twitch/api"
	newToken := getNewApiToken()
	if err := os.WriteFile(apiFile, []byte(newToken), 0644); err != nil {
		fmt.Println("Error writing API token to file:", err)
		os.Exit(1)

	}
}

func getApiToken() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Failed to get user home directory:", err)
		os.Exit(1)
	}

	apiFile := homeDir + "/.cache/twitch/api"
	content, err := os.ReadFile(apiFile)
	if err != nil {
		fmt.Println("Failed to read API token from file, generating a new one.")
		newToken := getNewApiToken()
		if err := os.WriteFile(apiFile, []byte(newToken), 0644); err != nil {
			fmt.Println("Error writing API token to file:", err)
			os.Exit(1)
		}
		return newToken
	}
	return string(content)
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

	return resp, err
}

func GetStreamData(endpoint string) ([]map[string]interface{}, error) {
	accessToken := getApiToken()

	resp, err := sendRequest(endpoint, accessToken)
	if err != nil || resp.StatusCode != http.StatusOK {
		writeNewToken()
		accessToken = getApiToken()
		resp, err = sendRequest(endpoint, accessToken)
		if err != nil || resp.StatusCode != http.StatusOK {
			fmt.Printf("Second request failed with status code %d: %v", resp.StatusCode, err)
			os.Exit(1)
		}
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
