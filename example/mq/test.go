package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

func main() {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               "pulsar://9.134.243.104:6650",
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
	}
	defer client.Close()
	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic: "my-topic",
	})
	if err != nil {
		log.Fatal(err)
	}

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            "my-topic",
		SubscriptionName: "my-sub",
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = producer.Send(context.Background(), &pulsar.ProducerMessage{
		Payload: []byte("hello"),
	})
	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	fmt.Println("Published message1")
	producer.Close()

	msg, err := consumer.Receive(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	consumer.Ack(msg)

	fmt.Printf("c Received message msgId: %#v -- content: %s\n", msg.ID(), string(msg.Payload()))
	m, err := consumer.Receive(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("c Received message msgId: %#v -- content: %s\n", m.ID(), string(m.Payload()))
	consumer.Ack(msg)

	consumer.Close()
}
