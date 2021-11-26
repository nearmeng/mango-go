package mq

import (
	"context"
	"sync"

	"github.com/nearmeng/mango-go/plugin/log"
)

type MessageChan chan Message
type MessageChanSlice []chan Message
type MQEventBus struct {
	subscribers map[string]MessageChanSlice
	sync        *sync.RWMutex
}

func NewMQEventBus() *MQEventBus {
	return &MQEventBus{subscribers: make(map[string]MessageChanSlice), sync: &sync.RWMutex{}}
}
func (pub *MQEventBus) Subscribe(topic string, messageChan MessageChan) {
	pub.sync.Lock()
	defer pub.sync.Unlock()
	if msgChanSlice, ok := pub.subscribers[topic]; ok {
		pub.subscribers[topic] = append(msgChanSlice, messageChan)
	} else {
		pub.subscribers[topic] = make(MessageChanSlice, 10)
	}
}
func (pub *MQEventBus) Publish(topic string, message Message) {
	pub.sync.RLock()
	defer pub.sync.RUnlock()
	for _, msgChan := range pub.subscribers[topic] {
		msgChan <- message
	}
}

func RunMQEventLoop(ctx context.Context, pub *MQEventBus, reader Reader) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Debug("poll....")
			message, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Error("receive failed,%v", err)
				continue
			}
			pub.Publish(message.Topic(), message)
		}
	}
}
