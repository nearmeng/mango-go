package l5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"
)

const (
	QOS_CMD_CHECK                  = 1  /*过载检测 已废弃*/
	QOS_CMD_UPDATE                 = 2  /*更新信息 已废弃*/
	QOS_CMD_TMCFG                  = 3  /*时间设置 已废弃*/
	QOS_CMD_TMCFG_DEL              = 4  /*时间设置取消 已废弃*/
	QOS_CMD_REQCFG                 = 5  /*访问量设置 已废弃*/
	QOS_CMD_REGCFG_DEL             = 6  /*访问量设置取消 已废弃*/
	QOS_CMD_LISTCFG                = 7  /*并发量设置 已废弃*/
	QOS_CMD_LISTCFG_DEL            = 8  /*并发量设置取消 已废弃*/
	QOS_CMD_TYPECFG                = 9  /*QOS运行模式设置 已废弃*/
	QOS_CMD_TYPECFG_DEL            = 10 /*取消QOS运行模式设置 已废弃*/
	QOS_CMD_REG_ROUTE              = 11 /*请求路由 已废弃*/
	QOS_CMD_CFG_ROUTE              = 12 /*配置路由*/
	QOS_CMD_DEL_ROUTE_NODE         = 13 /*删除指定路由结点*/
	QOS_CMD_OUT_ROUTE_TABLE        = 14 /*打印路由信息，已经废弃*/
	QOS_CMD_CALLER_UPDATE          = 15 /*更新主调信息*/
	QOS_CMD_PRE_ROUTE              = 16 /*预取路由信息 已废弃*/
	QOS_CMD_ROUTE_NTY              = 17 /*路由结果上报，单次*/
	QOS_CMD_GET_STAT               = 18 /*路由结果上报，多次*/
	QOS_CMD_UPDATE_STAT            = 19 /*更新状态*/
	QOS_CMD_CFG_CYC                = 20 /*配置参数 已废弃*/
	QOS_CMD_PRE_ROUTE_V2           = 21 /*批量预取路由信息*/
	QOS_CMD_SPECIFIC_ROUTE         = 22 /*获取有状态到机器的路由，以废弃*/
	QOS_CMD_BATCH_GET_ROUTE        = 23 /*批量获取路由*/
	QOS_CMD_BATCH_GET_ROUTE_VER    = 24 /*批量获取路由,带版本号*/
	QOS_CMD_CFG_ROUTE_WEIGHT       = 25 /*配置路由,带权重*/
	QOS_CMD_ADD_SNAME_SID          = 26 /*添加名字服务sid*/
	QOS_CMD_DEL_SNAME_SID          = 27 /*删除名字服务sid*/
	QOS_CMD_QUERY_SNAME            = 28 /*名字服务查询*/
	QOS_CMD_CALLER_UPDATE_BIT64    = 29 /*更新主调信息*/
	QOS_CMD_BATCH_GET_ROUTE_WEIGHT = 35 /*获取带权重的路由信息,主要用于一致性hash*/
)

const (
	QOS_RTN_OK               = 0      // success
	QOS_RTN_ACCEPT           = 1      // success (forward compatiblility)
	QOS_RTN_LROUTE           = 2      // success (forward compatiblility)
	QOS_RTN_TROUTE           = 3      // success (forward compatiblility)
	QOS_RTN_STATIC_ROUTE     = 4      // success (forward compatiblility)
	QOS_RTN_INITED           = 5      // success (forward compatiblility)
	QOS_RTN_OVERLOAD         = -10000 // sid overload, all ip:port of sid(modid,cmdid) is not available
	QOS_RTN_TIMEOUT          = -9999  // timeout
	QOS_RTN_SYSERR           = -9998  // error
	QOS_RTN_SENDERR          = -9997  // send error
	QOS_RTN_RECVERR          = -9996  // recv error
	QOS_MSG_INCOMPLETE       = -9995  // msg bad format (forward compatiblility)
	QOS_CMD_ERROR            = -9994  // cmd invalid (forward compatiblility)
	QOS_MSG_CMD_ERROR        = -9993  // msg cmd invalid (forward compatiblility)
	QOS_INIT_CALLERID_ERROR  = -9992  // init callerid error
	QOS_RTN_PARAM_ERROR      = -9991  // parameter error
	QOS_RTN_LOCAL_ERROR      = -9990  //api local internal error
	QOS_RTN_ASYNC_INIT_ERROR = -9989  //async api not init or init error
)

var (
	defaultEndian = binary.LittleEndian
	agentIp       = "127.0.0.1"
	agentPort     = 8888
	gVersion      = 40103
	// 给小一点,当机器上没有对应的结果的时候,服务不至于对前段超时
	// 当本机有数据之后,肯定远远小于100ms就拿到数据了
	gDialTimeout  = time.Millisecond * 100
	maxPacketSize = 1024 * 16
)

func init() {
	//腾讯云STKE，pod启动时 会传入 L5_AGENT_IP 环境变量
	if ip := os.Getenv("L5_AGENT_IP"); ip != "" {
		SetAgentAddress(ip, 8888)
	}
}

// SetAgentAddress 从外部设置l5 agent地址，兼容腾讯云STKE，本机没有agent，需要外部传入
func SetAgentAddress(ip string, port uint16) {
	agentIp = ip
	agentPort = int(port)
}

func dial(cmd int32, key int32, args ...interface{}) ([]byte, error) {
	var buf bytes.Buffer
	var head bytes.Buffer
	var err error
	for _, v := range args {
		switch v.(type) {
		case string:
			if _, err = buf.WriteString(v.(string)); err != nil {
				return nil, err
			}
		case []byte:
			if _, err = buf.Write(v.([]byte)); err != nil {
				return nil, err
			}
		default:
			if err = binary.Write(&buf, defaultEndian, v); err != nil {
				return nil, err
			}
		}
	}

	var c net.Conn
	c, err = net.DialTimeout("udp", fmt.Sprintf("%s:%d", agentIp, agentPort), gDialTimeout)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	c.SetDeadline(time.Now().Add(gDialTimeout))
	for _, v := range []int32{cmd, int32(20 + buf.Len()), int32(key), 0, int32(os.Getpid())} {
		if err := binary.Write(&head, defaultEndian, v); err != nil {
			return nil, err
		}
	}
	if _, err := c.Write(append(head.Bytes(), buf.Bytes()...)); err != nil {
		return nil, err
	}
	switch cmd {
	//no reply
	case QOS_CMD_GET_STAT, QOS_CMD_CALLER_UPDATE_BIT64:
		return nil, nil
	}

	result := make([]byte, maxPacketSize)
	if size, err := c.Read(result); err != nil {
		return nil, err
	} else {
		result = result[0:size]
	}
	code := defaultEndian.Uint32(result[12:16])
	if code > 5 || code < 0 {
		return nil, fmt.Errorf("ret error:%d", int32(code))
	} else {
		return result[20:], nil
	}
}
