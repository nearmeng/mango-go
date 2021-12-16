package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"

	_ "github.com/nearmeng/mango-go/example/statelesssvr/module"
	"github.com/nearmeng/mango-go/plugin"
	"github.com/nearmeng/mango-go/plugin/mq/pulsar"
	"github.com/nearmeng/mango-go/plugin/transport"
	"github.com/nearmeng/mango-go/server_base/app"
)

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

type eventTcp struct {
}

func (*eventTcp) OnOpened(conn transport.Conn) {
	fmt.Printf("get conn %s connect\n", conn.GetRemoteAddr().String())
}

func (*eventTcp) OnClosed(conn transport.Conn, active bool) {
	fmt.Printf("get conn %s closed active %t\n", conn.GetRemoteAddr().String(), active)

}

func (*eventTcp) OnData(conn transport.Conn, data []byte) {
	headerSize := binary.LittleEndian.Uint32(data)
	dataStr := string(data[4:])
	fmt.Printf("conn %s get header_size %d data %s\n", conn.GetRemoteAddr().String(), headerSize, dataStr)

	sendData := []byte(string("hello client"))
	sendHeaderSize := len(sendData)
	sendBuff := make([]byte, 4+sendHeaderSize)
	binary.LittleEndian.PutUint32(sendBuff[0:4], uint32(sendHeaderSize))

	copy(sendBuff[4:], sendData)

	_ = conn.Send(sendBuff)
}

func main() {

	server := app.NewServerApp("stateless_svr")

	err := server.Init()
	if err != nil {
		panic(err)
	}

	pulsarIns := plugin.GetPluginInst("mq", "pulsar").(*pulsar.PulsarClient)
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

	/*
		tcpIns := plugin.GetPluginInst("transport", "tcp").(*tcp.TcpTransport)

		tcpIns.Init(transport.Options{EventHandler: &eventTcp{}})

		server.Mainloop()

		tcpIns.Uninit()

		err = server.Fini()
		if err != nil {
			fmt.Printf("server fini failed")
		}
	*/

}
