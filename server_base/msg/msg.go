package msg

import (
	"errors"
	"sync"

	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/transport"
	"github.com/nearmeng/mango-go/proto/csproto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoimpl"
)

type ConnEventHandler func(conn transport.Conn)
type ClientMsgHandler func(conn transport.Conn, header *csproto.CSHead, msg proto.Message)
type ServerMsgHandler func()

type MsgHandlerMgr struct {
	mutex            sync.RWMutex
	connEventHandler map[int32]ConnEventHandler
	clientMsgHandler map[int32]ClientMsgHandler
	serverMsgHandler map[int32]ServerMsgHandler
}

const (
	CONN_EVENT_START = 0
	CONN_EVENT_STOP  = 1
)

var (
	msgHandlerMgr = &MsgHandlerMgr{
		connEventHandler: map[int32]ConnEventHandler{},
		clientMsgHandler: map[int32]ClientMsgHandler{},
		serverMsgHandler: map[int32]ServerMsgHandler{},
	}
)

func RegisterConnMsgHandler(eventType int32, handler ConnEventHandler) error {
	msgHandlerMgr.mutex.Lock()
	defer msgHandlerMgr.mutex.Unlock()

	_, ok := msgHandlerMgr.connEventHandler[eventType]
	if ok {
		return errors.New("already find event type")
	}

	msgHandlerMgr.connEventHandler[eventType] = handler
	return nil
}

func RegisterClientMsgHandler(msgid int32, handler ClientMsgHandler) error {
	msgHandlerMgr.mutex.Lock()
	defer msgHandlerMgr.mutex.Unlock()

	_, ok := msgHandlerMgr.clientMsgHandler[msgid]
	if ok {
		return errors.New("already find msgid")
	}

	msgHandlerMgr.clientMsgHandler[msgid] = handler
	return nil
}

func RegisterServerMsgHandler(msgid int32, handler ServerMsgHandler) error {
	msgHandlerMgr.mutex.Lock()
	defer msgHandlerMgr.mutex.Unlock()

	_, ok := msgHandlerMgr.serverMsgHandler[msgid]
	if ok {
		return errors.New("already find msgid")
	}

	msgHandlerMgr.serverMsgHandler[msgid] = handler
	return nil
}

func OnClientConnOpened(conn transport.Conn) {
	log.Info("client connect by connid %v", conn.GetConnID())

	msgHandlerMgr.connEventHandler[CONN_EVENT_START](conn)
}

func OnClientConnClosed(conn transport.Conn, active bool) {
	log.Info("client disconnnect of connid %v active %d", conn.GetConnID(), active)

	msgHandlerMgr.connEventHandler[CONN_EVENT_STOP](conn)
}

func printCSMsg(header *csproto.CSHead, msg proto.Message) {
	headerStr := protoimpl.X.MessageStringOf(header)
	msgStr := protoimpl.X.MessageStringOf(msg)

	log.Info("recv cs msg:\n %s\n %s", headerStr, msgStr)
}

func printSCMsg(header *csproto.SCHead, msg proto.Message) {
	headerStr := protoimpl.X.MessageStringOf(header)
	msgStr := protoimpl.X.MessageStringOf(msg)

	log.Info("recv sc msg:\n %s\n %s", headerStr, msgStr)
}

func RecvClientMsg(conn transport.Conn, data []byte) {
	header, msg := getCodec(CODEC_DEFAULT).Decode(data)
	if header == nil || msg == nil {
		log.Error("conn %v client msg decode failed", conn.GetConnID())
		return
	}

	printCSMsg(header, msg)

	h, ok := msgHandlerMgr.clientMsgHandler[header.GetMsgid()]
	if !ok {
		log.Error("msgid %d is not register", header.GetMsgid())
		return
	}

	h(conn, header, msg)
}

func SendToClient(conn transport.Conn, header *csproto.SCHead, msg proto.Message) error {
	data, err := getCodec(CODEC_DEFAULT).Encode(header, msg)
	if err != nil {
		log.Error("conn %v client msg encode failed", conn.GetConnID())
		return err
	}

	err = conn.Send(data)
	if err != nil {
		log.Error("conn %v client msg send failed", conn.GetConnID())
		return err
	}

	printSCMsg(header, msg)

	return nil
}

// by mosn
func RecvServerMsg(conn transport.Conn, data []byte) {

}

func SendToServer(data []byte) {

}
