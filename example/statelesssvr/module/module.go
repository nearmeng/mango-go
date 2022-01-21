package module

import (
	"github.com/nearmeng/mango-go/plugin"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/rpc/trpc"
	"github.com/nearmeng/mango-go/plugin/transport"
	"github.com/nearmeng/mango-go/proto/csproto"
	"github.com/nearmeng/mango-go/server_base/app"
	msgHandler "github.com/nearmeng/mango-go/server_base/msg"
	"google.golang.org/protobuf/proto"

	pb "github.com/nearmeng/mango-go/example/statelesssvr/proto/echo"
)

type TestModule struct {
	testValue int
}

func (m *TestModule) Init() error {

	m.testValue = 1

	r := plugin.GetPluginInst("rpc", "trpc").(*trpc.TrpcServer)
	pb.RegisterEchoService(r.GetServer(), &echoServiceImpl{})

	msgHandler.RegisterClientMsgHandler(int32(csproto.CSMessageID_cs_login), m.OnLogin)
	msgHandler.RegisterConnMsgHandler(msgHandler.CONN_EVENT_START, m.OnStart)
	msgHandler.RegisterConnMsgHandler(msgHandler.CONN_EVENT_STOP, m.OnStop)

	log.Info("test module init")

	return nil
}

func (m *TestModule) UnInit() error {
	log.Info("test module uninit")

	return nil
}

func (m *TestModule) Mainloop() {
}

func (m *TestModule) IsPreInit() bool {
	return true
}

func (m *TestModule) OnStart(conn transport.Conn) {
	log.Info("conn %d is start, remote %s", conn.GetConnID(), conn.GetRemoteAddr().String())
}

func (m *TestModule) OnStop(conn transport.Conn) {
	log.Info("conn %d is stop active %d, remote %s", conn.GetConnID(), conn.GetRemoteAddr().String())
}

func (m *TestModule) OnLogin(conn transport.Conn, header *csproto.CSHead, msg proto.Message) {
	log.Info("header msg_id %d seq_id %d", header.Msgid, header.Seqid)

	loginMsg := msg.(*csproto.CS_LOGIN)
	log.Info("login msg is %s", loginMsg.Name)

	rspHeader := &csproto.SCHead{
		Msgid: header.Msgid,
		Seqid: header.Seqid + 1,
	}

	rsp := &csproto.SC_LOGIN{
		Success: 1,
	}

	msgHandler.SendToClient(conn, rspHeader, rsp)
}

func (m *TestModule) GetName() string {
	return "test_module"
}

func (m *TestModule) OnReload() {
	log.Info("test module reload")
}

func init() {
	app.RegisterModule(&TestModule{})
}
