package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const (
	apiURL    = "https://api.twitch.tv/helix"
	idURL     = "https://id.twitch.tv/oauth2/token"
)

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

func getApiToken() string {
	apiKey := os.Getenv("TWITCH_API_KEY")
	apiSecret := os.Getenv("TWITCH_API_SECRET")

	if apiKey == "" || apiSecret == "" {
		fmt.Println("TWITCH_API_KEY or TWITCH_API_SECRET environment variable is not set")
		os.Exit(1)
	}

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

func readFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func writeFile(filePath, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

func GetStreamData(endpoint string) ([]map[string]interface{}, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory")
	}

	apiFile := homeDir + cacheDir + "/api"
	var accessToken string

	if content, err := readFile(apiFile); err == nil {
		accessToken = content
	} else {
		accessToken = getApiToken()
		if err := writeFile(apiFile, accessToken); err != nil {
			return nil, fmt.Errorf("error writing API token to file: %v", err)
		}
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Client-ID", apiKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("initial request failed with status code %d: %v", resp.StatusCode, err)
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
