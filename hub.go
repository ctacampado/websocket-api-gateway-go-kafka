package main

import (
	"log"

	uuid "github.com/satori/go.uuid"
)

type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	ecmsg      chan *EnrichedClientMessage
}

func newHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uuid.UUID]*Client),
		ecmsg:      make(chan *EnrichedClientMessage),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			var err error
			cid := uuid.Must(uuid.NewV4(), err)
			if err != nil {
				log.Printf("error generating uuid: %s", err)
				return
			}
			h.clients[cid] = client
			client.cid = cid
			log.Print("...connected client ", cid)
		case client := <-h.unregister:
			if nil != h.clients[client.cid] {
				log.Print("...disconnected client ", client.cid)
				prestr := "...removing client from hub..."
				delete(h.clients, client.cid)
				if nil != h.clients[client.cid] {
					log.Printf(prestr+"error: cannot remove client %s from hub!", client.cid)
				} else {
					log.Printf(prestr+"client %s removed!", client.cid)
				}
			}
		case ecmsg := <-h.ecmsg:
			log.Print(ecmsg.ClientMsg.Acctype)
			log.Printf("%+v\n", *h.clients[ecmsg.ClientID])
		}
	}
}
