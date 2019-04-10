package main

import (
	"encoding/json"
	"log"

	"github.com/Shopify/sarama"
	uuid "github.com/satori/go.uuid"
	"github.com/wvanbergen/kafka/consumergroup"
)

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	pmsg       chan *ClientMessage
	cmsg       chan *ConsumerMessage
	producer   sarama.SyncProducer
	ctopics    []string
	consumers  *consumergroup.ConsumerGroup
}

func newHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
		pmsg:       make(chan *ClientMessage),
		cmsg:       make(chan *ConsumerMessage),
		ctopics:    make([]string, 0),
	}
}

func clientRegistration(h *Hub) {
	for {
		select {
		case client := <-h.register:
			var err error
			cid := uuid.Must(uuid.NewV4(), err)
			if err != nil {
				log.Printf("error generating uuid: %s", err)
				return
			}
			h.clients[cid.String()] = client
			client.cid = cid.String()
			log.Print("...connected client ", cid)
			cmsg := &ClientMessage{}
			cmsg.CID = client.cid
			cmsg.OrgCode = client.Type
			log.Printf("cmsg %+v\n", cmsg)
			h.pmsg <- cmsg
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
		}
	}
}

func (h *Hub) run() {
	c := initConfig()
	//create consumers
	for _, element := range c.Topics.Consume {
		h.ctopics = append(h.ctopics, element)
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

	go clientRegistration(h)

	for {
		select {
		case msg := <-h.pmsg:
			log.Printf("msg from client %+v\n", msg)
			if nil != msg.Payload {
				if nil != msg.Payload.EnrollmentApproval {
					log.Printf("EnrollmentApproval: %+v\n", msg.Payload.EnrollmentApproval)
				}
			}
			var topic string
			if nil != msg.Payload {
				topic = "ledgertx.req"
			} else {
				topic = "clients.connected"
			}
			message, err := json.Marshal(msg)
			if err != nil {
				break
			}
			publish(message, topic, h.producer)
		case cmsg := <-h.cmsg:
			msg := &ClientMessage{}
			_ = json.Unmarshal([]byte(cmsg.Value), msg)
			log.Printf("msg %+v\n", msg)
			log.Printf("payload %+v\n", *msg.Payload)
			log.Printf("cid %s\n", msg.CID)
			client := h.clients[msg.CID]
			if nil != client {
				log.Printf("client %+v\n", client)
				log.Printf("type: %s\n", msg.Type)
				if nil != msg.Payload.EnrollmentApproval {
					log.Printf("EnrollmentApproval: %+v\n", msg.Payload.EnrollmentApproval)
				}
				err := client.conn.WriteJSON(*msg)
				if err != nil {
					log.Printf("error: %v", err)
					client.conn.Close()
					h.unregister <- client
					//delete(h.clients, msg.CID)
				}
			} else {
				for _, client := range h.clients {
					if nil != msg.Payload.Employee {
						log.Printf("fpcode %s\n", msg.Payload.Employee.FPInfo.FPCode)
						if client.Type == msg.Payload.Employee.FPInfo.FPCode {
							err := client.conn.WriteJSON(*msg)
							if err != nil {
								log.Printf("error: %v", err)
								client.conn.Close()
								//delete(h.clients, msg.CID)
								h.unregister <- client
							}
						}
					}

					if nil != msg.Payload.Block {
						err := client.conn.WriteJSON(*msg)
						if err != nil {
							log.Printf("error: %v", err)
							client.conn.Close()
							//delete(h.clients, msg.CID)
							h.unregister <- client
						}
					}

					if "" != msg.Type {
						if client.Type == msg.Type {
							err := client.conn.WriteJSON(*msg)
							if err != nil {
								log.Printf("error: %v", err)
								client.conn.Close()
								//delete(h.clients, msg.CID)
								h.unregister <- client
							}
						}
					}

				}
			}
		}
	}
}
