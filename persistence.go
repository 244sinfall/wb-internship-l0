package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nats-io/stan.go"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

func (p *Persistence) publish(amount int) {
	for i := 1; i <= amount; i++ {
		go func() {
			item, err := json.Marshal(generateFakeMessage())
			err = (*p.natsConnection).Publish(subjectName, item)
			if err != nil {
				fmt.Println("Error while publishing fake messages: " + err.Error())
			}
		}()
	}
}

func (p *Persistence) getItemFromDb(orderUid string) (Message, error) {
	var newMessage Message
	var deliveryPhone string
	var paymentId string
	var items []byte
	status := make(chan error)
	msg := make(chan Message)
	query := fmt.Sprintf("SELECT * FROM orders WHERE order_uid = '%s'", orderUid)
	jobs := sync.WaitGroup{}
	go func() {
		defer func() {
			close(status)
			close(msg)
		}()
		order := p.dbConnection.QueryRow(query)
		err := order.Scan(&newMessage.OrderUid, &newMessage.TrackNumber, &newMessage.Entry, &deliveryPhone, &paymentId, &newMessage.Locale,
			&newMessage.InternalSignature, &newMessage.CustomerId, &newMessage.DeliveryService, &newMessage.Shardkey,
			&newMessage.SmId, &newMessage.DateCreated, &newMessage.OofShard, &items)
		if len(items) != 0 {
			err := json.Unmarshal(items, &newMessage.Items)
			if err != nil {
				status <- errors.New("could not unmarshal items from order - " + err.Error())
			}
		}
		if newMessage.OrderUid != orderUid || err != nil {
			status <- errors.New("no match of db record with requested scheme or scan error (order) - " + err.Error())
		}
		jobs.Add(1)
		go func() {
			defer jobs.Done()
			query := fmt.Sprintf("SELECT * FROM deliveries WHERE phone = '%s'", deliveryPhone)
			delivery := p.dbConnection.QueryRow(query)
			err := delivery.Scan(&newMessage.Delivery.Phone, &newMessage.Delivery.Name, &newMessage.Delivery.Zip,
				&newMessage.Delivery.City, &newMessage.Delivery.Address, &newMessage.Delivery.Region,
				&newMessage.Delivery.Email)
			if newMessage.Delivery.Phone != deliveryPhone || err != nil {
				status <- errors.New("no match of db record with requested scheme or scan error (delivery) - " + err.Error())
			}
		}()
		jobs.Add(1)
		go func() {
			defer jobs.Done()
			query := fmt.Sprintf("SELECT * FROM payments WHERE transaction_id = '%s'", paymentId)
			payment := p.dbConnection.QueryRow(query)
			err := payment.Scan(&newMessage.Payment.Transaction, &newMessage.Payment.RequestId, &newMessage.Payment.Currency,
				&newMessage.Payment.Provider, &newMessage.Payment.Amount, &newMessage.Payment.PaymentDt,
				&newMessage.Payment.Bank, &newMessage.Payment.DeliveryCost, &newMessage.Payment.GoodsTotal,
				&newMessage.Payment.CustomFee)
			if newMessage.Payment.Transaction != paymentId || err != nil {
				status <- errors.New("no match of db record with requested scheme or scan error (payment) - " + err.Error())
			}
		}()
		jobs.Wait()
		msg <- newMessage
	}()
	for {
		select {
		case err := <-status:
			return Message{}, err
		case message := <-msg:
			p.cache.add(message)
			return message, nil
		}
	}
}

func (p *Persistence) publishModel() {
	jsonFile, err := os.Open("model.json")
	byteValue, err := ioutil.ReadAll(jsonFile)
	err = (*p.natsConnection).Publish(subjectName, byteValue)
	if err != nil {
		fmt.Println("Error while publishing model message: " + err.Error())
	}
}

func (p *Persistence) executeCommand(command string) error {
	switch {
	case command == "publishmodel":
		p.publishModel()
	case command == "publish":
		p.publish(1000)
	case command == "quit":
		p.closeSession()
	case command == "http":
		if p.isHTTPServerAlive() {
			p.closeHTTPServer()
			fmt.Println("HTTP server closed.")
		} else {
			p.startHTTPServer()
			fmt.Println("HTTP server connected.")
		}
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
		subscription, err := (*p.natsConnection).Subscribe(subjectName, p.handleMessage)
		if err != nil {
			fmt.Println("Subscription failed with error: " + err.Error())
			fmt.Println("Retry in 3 seconds...")
			time.Sleep(3 * time.Second)
			continue
		} else {
			p.natsSubscription = &subscription
			return
		}
	}

}

func (p *Persistence) closeSession() {
	err := (*p.natsSubscription).Unsubscribe()
	if err != nil {
		fmt.Println("Error unsubscribing from stream: " + err.Error())
	}
	fmt.Println("Stopped listening to messages...")
	err = p.dbConnection.Close()
	if err != nil {
		fmt.Println("Error closing DB connection: " + err.Error())
	}
	fmt.Println("DB connection closed...")
	err = (*p.natsConnection).Close()
	if err != nil {
		fmt.Println("Error closing nats-streaming connection: " + err.Error())
	}
	fmt.Println("nats-streaming connection closed...")
	if p.isHTTPServerAlive() {
		p.closeHTTPServer()
		fmt.Println("HTTP server closed...")
	}
	os.Exit(0)
}

func (p *Persistence) handleMessage(m *stan.Msg) {
	var newMessage Message
	err := json.Unmarshal(m.Data, &newMessage)
	if err != nil {
		fmt.Println("Error parsing message to given model: " + err.Error())
		return
	}
	go func() {
		query := fmt.Sprintf("CALL add_message('%s')", string(m.Data))
		_, _ = p.dbConnection.Exec(query)
	}()
	p.cache.add(newMessage)
	if err != nil {
		fmt.Println("Error adding message to cache: " + err.Error())
		return
	}
}

func (p *Persistence) closeHTTPServer() {
	if err := p.httpServer.Shutdown(context.TODO()); err != nil {
		fmt.Println("Error closing http server: " + err.Error())
	}
	p.httpWaitGroup.Wait()
}

func (p *Persistence) startHTTPServer() {

	fs := http.FileServer(http.Dir("frontend"))
	mux := http.NewServeMux()
	mux.Handle("/frontend/", http.StripPrefix("/frontend/", fs))
	mux.HandleFunc("/", p.showMainPage)
	mux.HandleFunc("/orders/", p.getItem)
	p.httpWaitGroup = &sync.WaitGroup{}
	p.httpServer = &http.Server{Addr: webServerIpAndPort, Handler: mux}
	p.httpWaitGroup.Add(1)
	go func() {
		defer p.httpWaitGroup.Done()
		if err := p.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Println("Error occured to HTTP connection: " + err.Error())
		}
	}()
}

func (p *Persistence) isHTTPServerAlive() bool {
	r, e := http.Head(webServerProtocol + webServerIpAndPort)
	return e == nil && r.StatusCode == 200
}

func (p *Persistence) setupNatsConnection() {
	for {
		connection, err := stan.Connect(natsClusterId, natsClientId)
		if err != nil {
			fmt.Println("Unable to start session with nats-streaming. Error: " + err.Error())
			fmt.Println("Will restart in 3 seconds...")
			time.Sleep(3 * time.Second)
			continue
		} else {
			p.natsConnection = &connection
			return
		}
	}
}

func (p *Persistence) fetchToCache() int {
	query := fmt.Sprintf("SELECT order_uid FROM orders ORDER BY date_created DESC LIMIT %d", maxCachedItems)
	orders, err := p.dbConnection.Query(query)
	if err != nil {
		fmt.Println("Error fetching orders to cache: " + err.Error())
	}
	for orders.Next() {
		var orderUid string
		err := orders.Scan(&orderUid)
		if err != nil {
			fmt.Println("Error fetching order uid from DB: " + err.Error())
			continue
		}
		_, err = p.getItemFromDb(orderUid)
		if err != nil {
			fmt.Println("Failed fetching and caching message: " + err.Error())
		}
	}
	return len(p.cache.messages)
}

func (p *Persistence) setupDbConnection() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		psqlHost, psqlPort, psqlUser, psqlPassword, psqlDatabase)
	for {
		dbConnection, err := sql.Open("postgres", psqlInfo)
		err = dbConnection.Ping()
		if err != nil {
			fmt.Println("Unable to start session with DB. Error: " + err.Error())
			fmt.Println("Will restart in 3 seconds...")
			time.Sleep(3 * time.Second)
			continue
		} else {
			p.dbConnection = dbConnection
			return
		}
	}
}

func startSession() Persistence {
	fmt.Println("Starting a session...")
	var persistence Persistence
	persistence.setupNatsConnection()
	fmt.Println("Nats connection established.")
	persistence.cache.messages = make(map[int]Message)
	fmt.Println("Cache object initialized.")
	persistence.setupDbConnection()
	fmt.Println("Database connected. Trying to fetch orders to cache...")
	fetched := persistence.fetchToCache()
	fmt.Printf("Fetched %d messages from the DB.\n", fetched)
	persistence.subscribe()
	fmt.Println("Subscribed successfully. Listening for messages...")
	return persistence
}
