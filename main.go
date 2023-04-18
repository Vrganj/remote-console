package main

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v2"
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

type Config struct {
	ApiUrl   string `yaml:"api-url"`
	ApiKey   string `yaml:"api-key"`
	ServerId string `yaml:"server-id"`
}

func main() {
	var c *websocket.Conn

	buf, err := ioutil.ReadFile("config.yml")

	if err != nil {
		log.Println("failed to read config.yml")
		return
	}

	var config Config

	if err = yaml.Unmarshal(buf, &config); err != nil {
		log.Println("failed to parse config")
		return
	}

	nologs := len(os.Args) >= 2 && os.Args[1] == "nologs"

	go func() {
		for {
			req, err := http.NewRequest("GET", config.ApiUrl+"/api/client/servers/"+config.ServerId+"/websocket", nil)

			if err != nil {
				panic(err)
			}

			req.Header.Add("Authorization", "Bearer "+config.ApiKey)

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
			h.Add("Origin", config.ApiUrl)

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

				if !nologs && message.Event == "console output" {
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
