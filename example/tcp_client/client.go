// Example function-based high-level Apache Kafka consumer
package main

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/nearmeng/mango-go/proto/csproto"
	"google.golang.org/protobuf/proto"
)

/**
 * Copyright 2016 Confluent Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// consumer_example implements a consumer using the non-channel Poll() API
// to retrieve messages and events.

func main() {
	/*
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

		c, err := kafka.NewConsumer(&kafka.ConfigMap{
			// Avoid connecting to IPv6 brokers:
			// This is needed for the ErrAllBrokersDown show-case below
			// when using localhost brokers on OSX, since the OSX resolver
			// will return the IPv6 addresses first.
			// You typically don't need to specify this configuration property.
			"bootstrap.servers":     "9.134.145.178:9092",
			"group.id":              "reader1",
			"broker.address.family": "v4",
			"session.timeout.ms":    6000,
			"auto.offset.reset":     "earliest"})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create consumer: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created Consumer %v\n", c)

		err = c.SubscribeTopics([]string{"test_kafka"}, nil)

		run := true

		for run {
			select {
			case sig := <-sigchan:
				fmt.Printf("Caught signal %v: terminating\n", sig)
				run = false
			default:
				ev := c.Poll(100)
				if ev == nil {
					continue
				}

				switch e := ev.(type) {
				case *kafka.Message:
					fmt.Printf("%% Message on %s:\n%s\n",
						e.TopicPartition, string(e.Value))
					if e.Headers != nil {
						fmt.Printf("%% Headers: %v\n", e.Headers)
					}
				case kafka.Error:
					// Errors should generally be considered
					// informational, the client will try to
					// automatically recover.
					// But in this example we choose to terminate
					// the application if all brokers are down.
					fmt.Fprintf(os.Stderr, "%% Error: %v: %v\n", e.Code(), e)
					if e.Code() == kafka.ErrAllBrokersDown {
						run = false
					}
				default:
					fmt.Printf("Ignored %v\n", e)
				}
			}
		}

		fmt.Printf("Closing consumer\n")
		c.Close()

	*/

	conn, err := net.Dial("tcp", "127.0.0.1:8888")
	if err != nil {
		fmt.Printf("connect server failed")
		return
	}

	defer conn.Close()

	header := &csproto.CSHead{
		Msgid: int32(csproto.CSMessageID_cs_login),
		Seqid: 1,
	}

	msg := &csproto.CS_LOGIN{
		Name: "test client",
		Sex:  "male",
		Account: &csproto.Account{
			Id:  100,
			Num: 10,
		},
	}

	//fmt.Printf("login full name %s\n", string(msg.ProtoReflect().Descriptor().FullName()))

	sendHeaderData, err := proto.Marshal(header)
	if err != nil {
		fmt.Printf("marshal failed")
		return
	}

	sendData, err := proto.Marshal(msg)
	if err != nil {
		fmt.Printf("marshal failed")
		return
	}

	sendBuff := make([]byte, 8+len(sendHeaderData)+len(sendData))
	sendHeaderSize := len(sendHeaderData)
	sendBodySize := len(sendData)

	binary.LittleEndian.PutUint32(sendBuff[0:4], uint32(sendHeaderSize))
	binary.LittleEndian.PutUint32(sendBuff[4:8], uint32(sendBodySize))
	//_ = append(sendBuff[8:], sendData...)
	copy(sendBuff[8:], sendHeaderData)
	copy(sendBuff[8+sendHeaderSize:], sendData)

	fmt.Printf("send header_size %d body_size %d\n", sendHeaderSize, sendBodySize)

	conn.Write(sendBuff)

	recvBuff := [512]byte{}
	n, _ := conn.Read(recvBuff[:])

	recvHeaderSize := binary.LittleEndian.Uint32(recvBuff[0:4])
	recvBodySize := binary.LittleEndian.Uint32(recvBuff[4:8])

	recvHeader := &csproto.SCHead{}
	err = proto.Unmarshal(recvBuff[8:8+recvHeaderSize], recvHeader)
	if err != nil {
		fmt.Printf("unmarshal failed")
		return
	}

	recvMsg := &csproto.SC_LOGIN{}
	err = proto.Unmarshal(recvBuff[8+recvHeaderSize:8+recvHeaderSize+recvBodySize], recvMsg)
	if err != nil {
		fmt.Printf("unmarshal failed")
		return
	}

	fmt.Printf("recv data from server header_size %d body_size %d read_len %d\n", recvHeaderSize, recvBodySize, n)
	fmt.Printf("header msgid %d seqid %d\n", recvHeader.Msgid, recvHeader.Seqid)
	fmt.Printf("msg sc login success %d\n", recvMsg.Success)

}
