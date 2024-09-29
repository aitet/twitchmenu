package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"net/http"
)

type Result struct {
	Type string
	Data []map[string]interface{}
	Err  error
}

const (
	namesFile  = "/.config/twitch/names"
	cacheDir   = "/.cache/twitch"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Failed to get user home directory")
		os.Exit(1)
	}

	namesFilePath := homeDir + namesFile
	cachePath := homeDir + cacheDir

	mkdir := exec.Command("mkdir", "-p", cachePath)
	if err := mkdir.Run(); err != nil {
		fmt.Printf("Error creating cache directory: %v\n", err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "-h":
			printHelp()
			return
		case "-e":
			openEditor(namesFilePath)
			return
		case "-l":
			showNames(namesFilePath)
			return
		case "-a":
			if len(args) > 1 {
				addName(namesFilePath, args[1])
			} else {
				fmt.Println("Usage: -a <name>")
			}
			return
		default:
			printHelp()
			return
		}
	}



	apiFile := homeDir + "/.cache/twitch/api"
	accessToken, err := getApiToken(apiFile)
	if err != nil {
		fmt.Printf("accsess token is weird: %v", err)
		os.Exit(1)
	}
	var wg sync.WaitGroup
	resultChannel := make(chan Result, 3)
	wg.Add(3)

	go func() {
		resp, testErr := sendRequest("/games/top?first=1", accessToken)
		if testErr != nil || resp.StatusCode != http.StatusOK {
			newToken, err := getNewApiToken(apiFile)
			if err != nil {
				return
			}
			accessToken = newToken
		}


		go func()  {
			defer wg.Done()
			top, err := GetStreamData("/streams?first=100", accessToken)
			resultChannel <- Result{Type: "top", Data: top, Err: err}
		}()

		go func() {
			defer wg.Done()
			games, err := GetStreamData("/games/top?first=100", accessToken)
			resultChannel <- Result{Type: "games", Data: games, Err: err}
		}()

		go func() {
			defer wg.Done()
			names, err := os.ReadFile(namesFilePath)
			if err != nil {
				resultChannel <- Result{Type: "followed", Err: err}
				return
			}
			streamers := strings.Fields(string(names))
			queryParams := []string{}
			for _, streamer := range streamers {
				if streamer != "" {
					queryParams = append(queryParams, "user_login="+streamer)
				}
			}
			followed, err := GetStreamData("/streams?" + strings.Join(queryParams, "&"), accessToken)
			resultChannel <- Result{Type: "followed", Data: followed, Err: err}
		}()


		go func() {
			wg.Wait()
			close(resultChannel)
		}()
	}()

	choices := []string{"top", "followed", "games"}
	choice := dmenu(choices, "-p twitch")
	if choice == "" {
		return
	}

	var top, followed, games []map[string]interface{}
	for result := range resultChannel {
		if result.Err != nil {
			fmt.Println("Response:", result.Err.Error())
			fmt.Println("Error:", result.Err)
			return
		}
		switch result.Type {
		case "top":
			top = result.Data
		case "games":
			games = result.Data
		case "followed":
			followed = result.Data
		}
		if followed != nil && top != nil && games != nil {
			break
		}
	}

	var streams []map[string]interface{}
	switch choice {
	case "top":
		streams = top
	case "followed":
		streams = followed
	case "games":
		gameNames := make([]string, len(games))
		for i, game := range games {
			gameNames[i] = game["name"].(string)
		}
		if selectedGame := dmenu(gameNames, "-l 20 -p games"); selectedGame != "" {
			for _, game := range games {
				if game["name"].(string) == selectedGame {
					send := "/streams?game_id=" + game["id"].(string)
					if streams, err = GetStreamData(send, accessToken); err != nil {
						fmt.Println("Error fetching streams:", err)
						return
					}
					break
				}
			}
		}
	}

	liveStreamers := []string{}
	for _, stream := range streams {
		liveStreamers = append(liveStreamers, fmt.Sprintf("%v\t%v", stream["viewer_count"], stream["user_login"]))
	}

	selectedStreamer := dmenu(liveStreamers, "-l 10 -p live")
	if selectedStreamer == "" {
		return
	}

	streamURL := "https://twitch.tv/" + strings.Split(selectedStreamer, "\t")[1]
	playStream(streamURL)
}

func printHelp() {
	fmt.Println(`Usage:
	twitch [ OPTION [...] ]

	Options:
	-a: 	Adds name to the list
	-e:	Opens the list in ${EDITOR:-vi}
	-l:	Show the list that will be checked
	-h:	Show help`)
}

func openEditor(filePath string) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error opening editor:", err)
	}
}

func showNames(filePath string) {
	names, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading names file:", err)
		return
	}
	fmt.Println(string(names))
}

func addName(filePath, name string) {
	names, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			os.WriteFile(filePath, []byte(name + "\n"), 0644)
			fmt.Println("Name added.")
			return
		}
		fmt.Println("Error reading names file:", err)
		return
	}
	if strings.Contains(string(names), name) {
		fmt.Println(name, "is already added.")
		return
	}

	err = os.WriteFile(filePath, append(names, []byte(name+"\n")...), 0644)
	if err != nil {
		fmt.Println("Error writing name to file:", err)
	} else {
		fmt.Println("Name added.")
	}
}

func dmenu(options []string, args string) string {
	cmd := exec.Command("dmenu", "-i", args)
	cmd.Stdin = strings.NewReader(strings.Join(options, "\n"))
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running dmenu:", err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(out))
}

func playStream(streamURL string) {
	player := "mpv"
	if _, err := exec.LookPath("streamlink"); err == nil {
		player = "streamlink"
	}
	cmd := exec.Command(player, streamURL)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error running player:", err)
	}
}
