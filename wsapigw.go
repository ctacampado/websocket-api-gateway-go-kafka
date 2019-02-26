package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wvanbergen/kafka/consumergroup"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 180 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 180 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type ClientMessage struct {
	CID      string   `json:"CID,omitempty"`
	Username string   `json:"Username,omitempty"`
	Type     string   `json:"Type,omitempty"`
	OrgCode  string   `json:"OrgCode,omitempty"`
	Payload  *Payload `json:"Payload,omitempty"`
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	cid  string
	Type string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var ip = flag.String("ip", "0.0.0.0", "http service address")
var port = flag.Int("port", 3000, "server port")

func handleClient(c *Client) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	//c.conn.SetReadLimit(maxMessageSize)
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
		log.Printf("...new msg!: %s\n", msg)
		cmsg := &ClientMessage{}
		_ = json.Unmarshal(msg, cmsg)
		if err != nil {
			log.Printf("error: %v", err)
			break
		}
		log.Printf("...new client msg!: %+v\n", cmsg)
		cmsg.CID = c.cid
		c.hub.pmsg <- cmsg
	}
}

func consumeKafka(h *Hub, cg *consumergroup.ConsumerGroup) {
	for {
		select {
		case msg := <-cg.Messages():
			// commit to zookeeper that message is read
			// this prevent read message multiple times after restart
			err := cg.CommitUpto(msg)
			if err != nil {
				log.Println("Error commit zookeeper: ", err.Error())
			}
			h.cmsg <- &ConsumerMessage{Value: string(msg.Value)}
		}
	}
}

func serveRA(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, Type: "RA"}
	hub.register <- client

	go handleClient(client)
	go consumeKafka(hub, hub.consumers)
}

func serveFPA(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, Type: "FPA"}
	hub.register <- client

	go handleClient(client)
	go consumeKafka(hub, hub.consumers)
}

func serveFPB(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, Type: "FPB"}
	hub.register <- client

	go handleClient(client)
	go consumeKafka(hub, hub.consumers)
}

func main() {
	flag.Parse()
	addr := *ip + ":" + strconv.Itoa(*port)
	hub := newHub()
	go hub.run()
	http.HandleFunc("/ra", func(w http.ResponseWriter, r *http.Request) {
		serveRA(hub, w, r)
	})
	http.HandleFunc("/fpa", func(w http.ResponseWriter, r *http.Request) {
		serveFPA(hub, w, r)
	})
	http.HandleFunc("/fpb", func(w http.ResponseWriter, r *http.Request) {
		serveFPB(hub, w, r)
	})
	log.Print("websocket server started! Now listening at ", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
