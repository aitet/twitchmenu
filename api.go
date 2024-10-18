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
	apiKey    = "cotxsalhlctv8z572f7fant4b0sc3u"
	apiSecret = "gaofxvult280l3sbz8n6btvk5fdswp"
)

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

func getNewApiToken(apiFile string) (string, error) {
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

	if err := os.WriteFile(apiFile, []byte(authResponse.AccessToken), 0644); err != nil {
		return "", fmt.Errorf("error writing API token to file: %v", err)
	}

	return authResponse.AccessToken, nil
}

func getApiToken(apiFile string) (string, error) {
	content, err := os.ReadFile(apiFile)
	if err != nil {
		newToken, err := getNewApiToken(apiFile)
		if err != nil {
			fmt.Printf("Failed to set new token: %v", err)
			os.Exit(1)
		}
		content = []byte(newToken)
	}

	return string(content), nil
}

func testRequest(accessToken string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL+"/games/top?first=1", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Client-ID", apiKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	_, err = client.Do(req)
	if err != nil {
		return err
	}
	return nil
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
		return nil, fmt.Errorf("Failed to send: %v", err)
	}

	return resp, nil
}

func GetStreamData(endpoint string, accessToken string) ([]map[string]interface{}, error) {
	resp, err := sendRequest(endpoint, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d: %v", resp.StatusCode, err)
	}

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
