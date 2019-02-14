package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"github.com/wvanbergen/kafka/consumergroup"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type ClientMessage struct {
	Clienttype string `json:"Clienttype"`
	Action     string `json:"Action"`
	Data       string `json:"Data"`
}

type EnrichedClientMessage struct {
	ClientID  uuid.UUID
	ClientMsg ClientMessage
}

type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	cid        uuid.UUID
	clienttype string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var ip = flag.String("ip", "localhost", "http service address")
var port = flag.Int("port", 8080, "server port")

func handleClient(c *Client) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		cmsg := &ClientMessage{}
		_ = json.Unmarshal(msg, cmsg)
		if err != nil {
			log.Printf("error: %v", err)
			break
		}
		log.Printf("...client msg: %+v\n", cmsg)
		c.clienttype = cmsg.Clienttype
		emsg := &EnrichedClientMessage{
			ClientID:  c.cid,
			ClientMsg: *cmsg}
		log.Printf("...enriched msg: %+v\n", emsg)

		c.hub.pmsg <- emsg
	}
}

func consumeKafka(h *Hub, cg *consumergroup.ConsumerGroup) {
	for {
		select {
		case msg := <-cg.Messages():
			// messages coming through chanel
			// only take messages from subscribed topic
			/*if msg.Topic != topic {
				continue
			}*/

			log.Println("Topic: ", msg.Topic)
			log.Println("Value: ", string(msg.Value))

			// commit to zookeeper that message is read
			// this prevent read message multiple times after restart
			err := cg.CommitUpto(msg)
			if err != nil {
				log.Println("Error commit zookeeper: ", err.Error())
			}
			h.cmsg <- &ConsumerMessage{Topic: string(msg.Topic), Value: string(msg.Value)}
		}
	}
}

func serve(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn}
	client.hub.register <- client

	go handleClient(client)
	go consumeKafka(hub, hub.consumers)
}

func main() {
	flag.Parse()
	addr := *ip + ":" + strconv.Itoa(*port)
	hub := newHub()
	go hub.run()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serve(hub, w, r)
	})
	log.Print("websocket server started! Now listening at ", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
