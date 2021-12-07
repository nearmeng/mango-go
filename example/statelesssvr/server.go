package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/nearmeng/mango-go/plugin"
	"github.com/nearmeng/mango-go/plugin/mq/implements/pulsar"
	"github.com/nearmeng/mango-go/server_base/app"
)

//_ "github.com/nearmeng/mango-go/example/statelesssvr/module"
//"github.com/nearmeng/mango-go/plugin/mq/implements/kafka"

type EventMessage struct {
	UserId    int64
	FightId   int64
	EventType string
	AttackSum int64
}

func (e *EventMessage) Partition() int {
	panic("implement me")
}

func (e *EventMessage) Key() string {
	return strconv.FormatInt(e.UserId, 10)
}

func (e *EventMessage) MetaInfo() map[string]string {
	return map[string]string{
		"user_id":    strconv.FormatInt(e.UserId, 10),
		"fight_id":   strconv.FormatInt(e.FightId, 10),
		"event_type": e.EventType,
	}
}

func (e *EventMessage) PayLoad() []byte {
	data, _ := json.Marshal(e)
	return data
}

func (e *EventMessage) SeqID() int64 {
	return 0
}

func (e *EventMessage) SetSeqID(seqID int64) {
	panic("implement me")
}

func (e *EventMessage) Topic() string {
	return "test_kafka"
}

func main() {

	server := app.NewServerApp("stateless_svr")

	err := server.Init()
	if err != nil {
		panic(err)
	}

	pulsarIns := plugin.GetPluginInst("mq", "pulsar").(*pulsar.PulsarIns)
	kreader := pulsarIns.GetReader("reader1")
	if kreader == nil {
		fmt.Printf("reader is nil")
		return
	}

	kwriter := pulsarIns.GetWriter("writer1")
	if kwriter == nil {
		fmt.Printf("writer is nil")
	}

	ctx := context.Background()

	_, err = kwriter.WriteMessage(ctx, &EventMessage{
		UserId:    1,
		FightId:   2,
		EventType: "type_b",
		AttackSum: 10,
	})
	if err != nil {
		fmt.Printf("writer failed for %v", err)
		return
	}

	m1, err := kreader.ReadMessage(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("read msg %v\n", m1)

	kreader.Ack(ctx, m1)
	fmt.Printf("ack finished\n")

	kreader.Close()
	fmt.Printf("reader closed\n")

	server.Mainloop()

	err = server.Fini()
	if err != nil {
		fmt.Printf("server fini failed")
	}

}
