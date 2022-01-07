package attalog

import (
	"fmt"
	"runtime"
	"time"

	"git.code.oa.com/atta/attaapi-go"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/metrics"
	"go.uber.org/zap/buffer"
)

// logConsumer 日志消费者
type logConsumer struct {
	Atta      *attaapi.AttaApi
	AttaID    string
	AttaToken string

	EnableBatch    bool
	SendInternalMs int
}

// logConsume 消费管道数据，实时发送atta
func (c *logConsumer) logConsume(ch chan *buffer.Buffer) {
	defer func() {
		// 捕获panic
		if err := recover(); err != nil {
			buf := make([]byte, 1024)
			buf = buf[:runtime.Stack(buf, false)]
			log.Errorf("trpc-log-atta: panic recovery, err:%v\n%s\n", err, buf)
		}
	}()

	for {
		c.SingleConsume(ch)
	}
}

// SingleConsume 消费管道数据
func (c *logConsumer) SingleConsume(ch chan *buffer.Buffer) error {
	// 消费者阻塞
	msg := <-ch
	defer msg.Free()

	ret := c.Atta.SendBinary(c.AttaID, c.AttaToken, msg.Bytes())
	if ret != 0 {
		metrics.Counter("AttaLoggerConsumerSendFail").Incr()
		return fmt.Errorf("atta send fail, ret:%d", ret)
	}

	return nil
}

// logConsumeRegular 周期消费管道数据，汇聚后批量发送atta
func (c *logConsumer) logConsumeRegular(ch chan *buffer.Buffer) {
	timer := time.NewTimer(time.Duration(c.SendInternalMs) * time.Millisecond)
	defer func() {
		timer.Stop()
		// 捕获panic
		if err := recover(); err != nil {
			buf := make([]byte, 1024)
			buf = buf[:runtime.Stack(buf, false)]
			log.Errorf("trpc-log-atta: panic recovery, err:%v\n%s\n", err, buf)
		}
	}()

	for {
		select {
		case <-timer.C: // 周期执行
			c.BatchConsume(ch)
			timer.Reset(time.Duration(c.SendInternalMs) * time.Millisecond)
		}
	}
}

// BatchConsume 批量发送日志
func (c *logConsumer) BatchConsume(ch chan *buffer.Buffer) error {
	var msgs []*buffer.Buffer
	length := 0

	for {
		var msg *buffer.Buffer
		isChEmpty := false

		select {
		case msg = <-ch:
		default:
			isChEmpty = true
		}

		if (msg != nil && length+msg.Len() > logMaxLength) || (isChEmpty && length > 0) {
			c.BatchSend2Atta(msgs)

			msgs = msgs[0:0]
			length = 0
		}

		if isChEmpty {
			break
		}

		msgs = append(msgs, msg)
		length += msg.Len()
	}
	return nil
}

// BatchSend2Atta 批量发送到atta
func (c *logConsumer) BatchSend2Atta(msgs []*buffer.Buffer) error {
	defer func() {
		for _, m := range msgs {
			m.Free()
		}
	}()

	var data [][]byte
	for _, m := range msgs {
		data = append(data, m.Bytes())
	}

	ret := c.Atta.BatchSendBinary(c.AttaID, c.AttaToken, data)
	if ret != 0 {
		metrics.Counter("AttaLoggerConsumerSendFail").Incr()
		return fmt.Errorf("atta send fail:%v", ret)
	}

	return nil
}

// start 开启log consumer
func (c *logConsumer) start(ch chan *buffer.Buffer) {
	if c.EnableBatch {
		go c.logConsumeRegular(ch)
	} else {
		go c.logConsume(ch)
	}
}
