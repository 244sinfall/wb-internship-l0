package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nats-io/stan.go"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func (p *Persistence) publish(amount int) {
	for i := 1; i <= amount; i++ {
		go func() {
			item, err := json.Marshal(generateFakeMessage())
			err = p.natsConnection.Publish(subjectName, item)
			if err != nil {
				fmt.Println("Error while publishing fake messages: " + err.Error())
			}
		}()
	}
}

func (p *Persistence) publishModel() {
	jsonFile, err := os.Open("model.json")
	byteValue, err := ioutil.ReadAll(jsonFile)
	err = p.natsConnection.Publish(subjectName, byteValue)
	if err != nil {
		fmt.Println("Error while publishing model message: " + err.Error())
	}
}

func (p *Persistence) executeCommand(command string) error {
	switch {
	case command == "publishmodel":
		p.publishModel()
	case command == "publish":
		p.publish(10)
	case command == "quit":
		p.closeSession()
	case command == "http":
		p.switchHttpServer()
	case command == "help":
		fmt.Println("publish [amount] - generate and publish [amount] random messages to stream")
		fmt.Println("checkpsql - return numbers of records stored in psql")
		fmt.Println("fakebreak - imitate service breakdown with data restore")
		fmt.Println("http - runs/stops http server to get data from cache")
		fmt.Println("quit - exit session")
	default:
		return errors.New("command does not exist. Check 'help' for more info")
	}
	return nil
}

func (p *Persistence) subscribe() {
	for {
		var err error
		p.natsSubscription, err = p.natsConnection.Subscribe(subjectName, p.handleMessage)
		if err != nil {
			fmt.Println("Subscription failed with error: " + err.Error())
			fmt.Println("Retry in 3 seconds...")
			time.Sleep(3 * time.Second)
			continue
		}
		fmt.Println("Subscribed successfully. Listening for messages...")
		return
	}

}

func (p *Persistence) closeSession() {
	err := p.natsSubscription.Unsubscribe()
	if err != nil {
		fmt.Println("Error unsubscribing from stream: " + err.Error())
	}
	fmt.Println("Stopped listening to messages...")
	err = p.dbConnection.Close()
	if err != nil {
		fmt.Println("Error closing DB connection: " + err.Error())
	}
	fmt.Println("DB connection closed...")
	err = p.natsConnection.Close()
	if err != nil {
		fmt.Println("Error closing nats-streaming connection: " + err.Error())
	}
	fmt.Println("nats-streaming connection closed...")
	os.Exit(0)
}

func (p *Persistence) handleMessage(m *stan.Msg) {
	var newMessage Message
	err := json.Unmarshal(m.Data, &newMessage)
	p.cache.add(newMessage)
	if err != nil {
		return
	}
}

func (p *Persistence) switchHttpServer() {
	p.httpServer.Addr = "127.0.0.1:3000"
	fs := http.FileServer(http.Dir("frontend"))
	go func() {
		http.Handle("/frontend/", http.StripPrefix("/frontend/", fs))
		http.HandleFunc("/", p.showMainPage)
		http.HandleFunc("/orders/", p.getItem)
		err := p.httpServer.ListenAndServe()
		if err != nil {

		}
	}()

}

func startSession() Persistence {
	fmt.Println("Starting a session...")
	var persistence Persistence
	var err error
	for {
		persistence.natsConnection, err = stan.Connect(natsClusterId, natsClientId)
		if err != nil {
			fmt.Println("Unable to start session with nats-streaming. Error: " + err.Error())
			fmt.Println("Will restart in 3 seconds...")
			time.Sleep(3 * time.Second)
			continue
		}
		psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			psqlHost, psqlPort, psqlUser, psqlPassword, psqlDatabase)
		persistence.dbConnection, err = sql.Open("postgres", psqlInfo)
		err = persistence.dbConnection.Ping()
		if err != nil {
			fmt.Println("Unable to start session with DB. Error: " + err.Error())
			fmt.Println("Will restart in 3 seconds...")
			time.Sleep(3 * time.Second)
			continue
		}
		persistence.cache.messages = make(map[int]Message)
		fmt.Println("Connection established. Session started...")
		return persistence
	}
}
