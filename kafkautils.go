package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kafka/consumergroup"
)

type Topic struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

type Topics struct {
	Produce []Topic `json:"produce"`
	Consume []Topic `json:"consume"`
}

type KafkaConfig struct {
	KafkaAddr     string `json:"kafka"`
	ZookeeperAddr string `json:"zookeeper"`
	Topics        Topics `json:"topics"`
	Cgroup        string `json:"cgroup"`
}

func initConfig() *KafkaConfig {
	c, err := os.Open("config.json")
	if err != nil {
		log.Print(err)
	}
	defer c.Close()

	byteValue, _ := ioutil.ReadAll(c)

	var result KafkaConfig
	json.Unmarshal([]byte(byteValue), &result)

	return &result
}

func initProducer(kaddr string) (sarama.SyncProducer, error) {
	log.Print(kaddr)
	// setup sarama log to stdout
	sarama.Logger = log.New(os.Stdout, "", log.Ltime)

	// producer config
	config := sarama.NewConfig()
	config.Producer.Retry.Max = 5
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true

	// async producer
	//prd, err := sarama.NewAsyncProducer([]string{kafkaConn}, config)

	// sync producer
	prd, err := sarama.NewSyncProducer([]string{kaddr}, config)

	return prd, err
}

func initConsumer(topic string, zaddr string, cgroup string) (*consumergroup.ConsumerGroup, error) {
	// consumer config
	config := consumergroup.NewConfig()
	config.Offsets.Initial = sarama.OffsetOldest
	config.Offsets.ProcessingTimeout = 2 * time.Second

	// join to consumer group
	log.Print("before joining consumer group!")
	cg, err := consumergroup.JoinConsumerGroup(cgroup, []string{topic}, []string{zaddr}, config)
	if err != nil {
		return nil, err
	}
	log.Print("joined consumer group!")
	return cg, err
}
