package main

import (
	"encoding/json"
	"log"

	"github.com/Shopify/sarama"
	"github.com/satori/go.uuid"
	"github.com/wvanbergen/kafka/consumergroup"
)

type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	pmsg       chan *EnrichedClientMessage
	cmsg       chan *ConsumerMessage
	producer   sarama.SyncProducer
	ptopics    map[string]string
	ctopics    []string
	consumers  *consumergroup.ConsumerGroup
}

func newHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uuid.UUID]*Client),
		pmsg:       make(chan *EnrichedClientMessage),
		cmsg:       make(chan *ConsumerMessage),
		ptopics:    make(map[string]string),
		ctopics:    make([]string, 0),
	}
}

func (h *Hub) run() {
	c := initConfig()
	//create consumers and producers
	for _, element := range c.Topics.Produce {
		h.ptopics[element.Action] = element.Name
	}
	for _, element := range c.Topics.Consume {
		h.ctopics = append(h.ctopics, element.Name)
	}
	prod, err := initProducer(c.KafkaAddr)
	if err != nil {
		log.Printf("error initializing producer: %s", err)
		return
	}
	h.producer = prod
	cons, err := initConsumer(h.ctopics, c.ZookeeperAddr, c.Cgroup)
	if err != nil {
		log.Printf("error initializing consumer: %s", err)
		return
	}
	h.consumers = cons

	log.Print("end init")

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
		case msg := <-h.pmsg:
			topic := h.ptopics[msg.ClientMsg.Action]
			message, err := json.Marshal(msg)
			if err != nil {
				break
			}
			publish(message, topic, h.producer)
		case cmsg := <-h.cmsg:
			log.Printf("%+v\n", cmsg)
		}
	}
}
