package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

type WsCredentials struct {
	Data struct {
		Token  string `json:"token"`
		Socket string `json:"socket"`
	} `json:"data"`
}

type DataMessage struct {
	Event string   `json:"event"`
	Args  []string `json:"args"`
}

func main() {
	var c *websocket.Conn

	apiUrl := flag.String("url", "https://control.heavynode.com", "The API URL")
	apiKey := flag.String("key", "", "Your Pterodactyl API key")
	serverId := flag.String("server", "", "Your server ID (get it from the network tab)")
	noLogs := flag.Bool("nologs", false, "Don't print server logs")
	flag.Parse()

	if *apiKey == "" {
		panic("Missing API key")
	}

	if *serverId == "" {
		panic("Missing server ID")
	}

	go func() {
		for {
			req, err := http.NewRequest("GET", *apiUrl+"/api/client/servers/"+*serverId+"/websocket", nil)

			if err != nil {
				panic(err)
			}

			req.Header.Add("Authorization", "Bearer "+*apiKey)

			res, err := http.DefaultClient.Do(req)

			if err != nil {
				panic(err)
			}

			buf, err := io.ReadAll(res.Body)

			if err != nil {
				panic(err)
			}

			var credentials WsCredentials
			err = json.Unmarshal(buf, &credentials)

			if err != nil {
				panic(err)
			}

			log.Println("got ws url: " + credentials.Data.Socket)

			h := http.Header{}
			h.Add("Origin", *apiUrl)

			c, _, err = websocket.DefaultDialer.Dial(credentials.Data.Socket, h)

			if err != nil {
				panic(err.Error())
			}

			message := DataMessage{
				Event: "auth",
				Args:  []string{credentials.Data.Token},
			}

			err = c.WriteJSON(message)

			if err != nil {
				panic(err)
			}

			for {
				_, buf, err := c.ReadMessage()

				if err != nil {
					panic(err)
				}

				var message DataMessage
				err = json.Unmarshal(buf, &message)

				if err != nil {
					panic(err)
				}

				if !*noLogs && message.Event == "console output" {
					log.Println(message.Args[0])
				} else if message.Event == "token expiring" {
					break
				}
			}

			c.Close()
		}
	}()

	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		message := DataMessage{
			Event: "send command",
			Args:  []string{text},
		}

		if err := c.WriteJSON(message); err != nil {
			panic(err)
		}
	}
}
