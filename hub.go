package main

import (
	"log"

	"github.com/Shopify/sarama"
	"github.com/satori/go.uuid"
	"github.com/wvanbergen/kafka/consumergroup"
)

type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	ecmsg      chan *EnrichedClientMessage
	producers  map[string]sarama.SyncProducer
	consumers  map[string]*consumergroup.ConsumerGroup
}

func newHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uuid.UUID]*Client),
		ecmsg:      make(chan *EnrichedClientMessage),
		producers:  make(map[string]sarama.SyncProducer),
		consumers:  make(map[string]*consumergroup.ConsumerGroup),
	}
}

func (h *Hub) run() {
	//initConfig
	c := initConfig()
	//create consumers and producers
	/*
		for _, element := range c.Topics.Produce {
			p, err := initProducer(c.KafkaAddr)
			if err != nil {
				log.Printf("error initializing producer: %s", err)
				return
			}
			h.producers[element.Name] = p
		}*/
	for _, element := range c.Topics.Consume {
		c, err := initConsumer(element.Name, c.ZookeeperAddr, c.Cgroup)
		if err != nil {
			log.Printf("error initializing consumer: %s", err)
			return
		}
		h.consumers[element.Name] = c
		log.Print(element.Name)
	}

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
		case ecmsg := <-h.ecmsg:
			log.Print(ecmsg.ClientMsg.Acctype)
			log.Printf("%+v\n", *h.clients[ecmsg.ClientID])
		}
	}
}
