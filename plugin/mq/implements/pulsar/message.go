package pulsar

import (
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
)

type pulsarMessage struct {
	msg pulsar.Message
}

func (p *pulsarMessage) String() string {
	return fmt.Sprintf("%s[%v]@%s", p.msg.Topic(), p.Key(), p.msg.ID())
}

func (p *pulsarMessage) Partition() int {
	panic("implement me")
}

func (p *pulsarMessage) MessageID() pulsar.MessageID {
	return p.msg.ID()
}

func (p *pulsarMessage) Key() string {
	return p.msg.Key()
}

func (p *pulsarMessage) MetaInfo() map[string]string {
	return p.msg.Properties()
}

func (p *pulsarMessage) PayLoad() []byte {
	return p.msg.Payload()
}

func (p *pulsarMessage) SeqID() int64 {
	return 0
}

func (p *pulsarMessage) SetSeqID(seqID int64) {
	return
}

func (p *pulsarMessage) Topic() string {
	return p.msg.Topic()
}

type producerMessage struct {
	m *pulsar.ProducerMessage
}

func (p *producerMessage) Partition() int {
	panic("implement me")
}

func (p *producerMessage) Key() string {
	return p.m.Key
}

func (p *producerMessage) MetaInfo() map[string]string {
	return p.m.Properties
}

func (p *producerMessage) PayLoad() []byte {
	return p.PayLoad()
}

func (p *producerMessage) SeqID() int64 {
	return *p.m.SequenceID
}

func (p *producerMessage) SetSeqID(seqID int64) {
	*p.m.SequenceID = seqID
}

func (p *producerMessage) Topic() string {
	return p.Topic()
}
