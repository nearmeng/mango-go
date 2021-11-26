package event

import (
	"fmt"
	"sync"
	"time"

	"github.com/nearmeng/mango-go/plugin/log"
)

const (
	_maxTopicSubscriberNum = 1024
	_maxTopicTimeout       = 1 * time.Second
)

type Subscriber struct {
	name     string
	callback func(i interface{}, ch chan struct{})
}

type Topic struct {
	timeout     time.Duration
	subscribers []Subscriber
}

type Publisher struct {
	m      sync.RWMutex
	topics map[string]*Topic
}

var (
	_publisher = &Publisher{
		topics: make(map[string]*Topic),
	}
)

func NewTopic(name string, timeout time.Duration) error {
	_publisher.m.Lock()
	defer _publisher.m.Unlock()

	_, ok := _publisher.topics[name]
	if ok {
		return fmt.Errorf("topic %s has already exist", name)
	}

	if timeout > _maxTopicTimeout {
		return fmt.Errorf("topic %s time is too max %v", name, timeout)
	}

	topic := &Topic{
		timeout:     timeout,
		subscribers: []Subscriber{},
	}

	_publisher.topics[name] = topic
	return nil
}

func RegisterSubscriber(name string, sub Subscriber) error {
	_publisher.m.Lock()
	defer _publisher.m.Unlock()

	topic, ok := _publisher.topics[name]
	if !ok {
		return fmt.Errorf("topic %s not create", name)
	}

	if len(topic.subscribers) >= _maxTopicSubscriberNum {
		return fmt.Errorf("topic %s subscriber too many", name)
	}

	for _, s := range topic.subscribers {
		if s.name == sub.name {
			return fmt.Errorf("subscriber %s has already subscriber topic %s", sub.name, name)
		}
	}

	topic.subscribers = append(topic.subscribers, sub)

	log.Info("topic %s add subcriber %s, total num %d", name, sub.name, len(topic.subscribers))
	return nil

}

func SendTopic(sub Subscriber, i interface{}, topic *Topic, wg *sync.WaitGroup) {
	defer wg.Done()

	ch := make(chan struct{})
	defer close(ch)

	go sub.callback(i, ch)
	if topic.timeout > 0 {
		select {
		case <-time.After(topic.timeout):
		case <-ch:
		}
	}
}

func Publish(name string, i interface{}) error {
	_publisher.m.Lock()
	defer _publisher.m.Unlock()

	topic, ok := _publisher.topics[name]
	if !ok {
		return fmt.Errorf("topic %s not create", name)
	}

	var wg sync.WaitGroup
	for _, sub := range topic.subscribers {
		wg.Add(1)
		go SendTopic(sub, i, topic, &wg)
	}

	wg.Wait()
	return nil
}
