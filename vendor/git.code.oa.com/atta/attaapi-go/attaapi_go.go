package attaapi

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const attaAgentUDPHost string = "127.0.0.1"                            // 本机，暂时不支持给其他机器发UDP包
const attaAgentUDPPort int = 6588                                      // 默认的UDP端口号
const attaAgentUnixPath string = "/data/pcgatta/agent/atta_agent.unix" //默认Unix套接字路径
const attaAgentUDPPorts string = "6588,16588,26588,36588,46588,56588,9112,19112,29112,39112,49112,59112" +
	",10015,20015,30015,40015,50015,60015"

const AttaReportCodeSuccess int = 0         // 成功
const AttaReportCodeInvalidMsg int = -1     //消息无效，可能原因有消息空指针、消息长度为0、消息字段列表为空、批量发送消息列表为空
const AttaReportCodeOverLimit int = -2      // 消息超长
const AttaReportCodeNetFailed int = -3      //发送失败，可能原因有UDP系统缓存满了、发送过程中socket连接被中断、UDP发送消息长度+Atta消息头长度超过UDP协议约束64K-20
const AttaReportCodeNotInit = -4            //未初始化，没有调用init初始化函数
const AttaReportCodeNoUsablePort = -5       //无可用端口
const AttaReportCodeCreateSocketFailed = -6 //创建Socket失败
const AttaReportCodeConnetSocketFailed = -7 //连接Socket失败
const AttaReportCodeUnknownSocketType = -8  //socket类型未知
const AttaReportCodeAgentBusy = -9          //agent 繁忙
const AttaReportCodeTCPWriteFailed = -10    //写 TCP 失败（超时）
const AttaReportCodeTCPReadFailed = -11     //读 TCP 失败（超时）
const AttaReportCodeOverBatchNum = -12      //批量上报数据条数超过上限

const AgentReturnCodeBusy = 10 //agent 繁忙

//上报均使用此结构体
type AttaApi struct {
	m_type       int
	m_ip         string
	m_port       string
	m_path       string
	m_localip    string
	conn         net.Conn
	obj          unsafe.Pointer
	lock         sync.Mutex
	initState    int32
	conState     int32
	reConGrState int32
	reConTime    time.Time
	alarmGrState int32
	alarmTime    time.Time
	alarmAttaId  string
	rwTimeout    int
}

//初始化UDP通讯，每个实例只需要调用一次
//返回:0 成功, 其它 失败
func (p *AttaApi) InitUDP() int {
	return p.init(socketTypeUdp)
}

//初始化TCP通讯，每个实例只需要调用一次
//返回:0 成功, 其它 失败
func (p *AttaApi) InitTCP() int {
	return p.init(socketTypeTcp)
}

func (p *AttaApi) SetTCPRWTimeout(timeOut int) {
	if timeOut < rwTimeoutMin || timeOut > rwTimeoutMax {
		p.rwTimeout = defaultRWTimeout
	} else {
		p.rwTimeout = timeOut
	}
}

//初始化Unix套接字通讯，每个实例只需要调用一次
//返回:0 成功, 其它 失败
func (p *AttaApi) InitUnix() int {
	return p.init(socketTypeUnix)
}

//发送string slice
//参数:sAttaid Attaid, sToken Token，详见attaid 信息；autoEscape（true 自动转义，false 不自动转义）。
//返回:0 成功, 其它 失败
//注意:选择自动转义时会有一定的性能消耗。
func (p *AttaApi) SendFields(sAttaid string, sToken string, _fields []string, autoEscape bool) int {
	temp := make([][]string, 1)
	temp[0] = _fields
	if autoEscape {
		return p.sendStringFieldsAutoEscape(sAttaid, sToken, temp, sourceAtta)
	} else {
		return p.sendStringFields(sAttaid, sToken, temp, sourceAtta)
	}
}

//发送string
//参数:sAttaid Attaid, sToken Token，详见attaid 信息
//返回:0 成功, 其它 失败
func (p *AttaApi) SendString(sAttaid string, sToken string, sData string) int {
	temp := make([]string, 1)
	temp[0] = sData
	result := p.sendString(sAttaid, sToken, temp, sourceAtta)
	return result
}

//发送byte slice
//参数:sAttaid Attaid, sToken Token，详见attaid 信息
//返回:0 成功, 其它 失败
func (p *AttaApi) SendBinary(sAttaid string, sToken string, bData []byte) int {
	temp := make([][]byte, 1)
	temp[0] = bData
	return p.sendBinary(sAttaid, sToken, temp, sourceAtta)
}

//批量发送string slice
//参数:sAttaid Attaid, sToken Token，详见attaid 信息；autoEscape（true 自动转义，false 不自动转义）。
//返回:0 成功, 其它 失败
//注意:选择自动转义时会有一定的性能消耗。
func (p *AttaApi) BatchSendFields(sAttaid string, sToken string, _fields [][]string, autoEscape bool) int {
	rec := p.checkDataNum(len(_fields))
	if rec != AttaReportCodeSuccess {
		return rec
	}
	if autoEscape {
		return p.sendStringFieldsAutoEscape(sAttaid, sToken, _fields, sourceMAtta)
	} else {
		return p.sendStringFields(sAttaid, sToken, _fields, sourceMAtta)
	}
}

//批量发送string
//参数:sAttaid Attaid, sToken Token，详见attaid 信息
//返回:0 成功, 其它 失败
func (p *AttaApi) BatchSendString(sAttaid string, sToken string, sData []string) int {
	rec := p.checkDataNum(len(sData))
	if rec != AttaReportCodeSuccess {
		return rec
	}
	result := p.sendString(sAttaid, sToken, sData, sourceMAtta)
	return result
}

//批量发送byte slice
//参数:sAttaid Attaid, sToken Token，详见attaid 信息
//返回:0 成功, 其它 失败
func (p *AttaApi) BatchSendBinary(sAttaid string, sToken string, bData [][]byte) int {
	rec := p.checkDataNum(len(bData))
	if rec != AttaReportCodeSuccess {
		return rec
	}
	return p.sendBinary(sAttaid, sToken, bData, sourceMAtta)
}

/**
 * 将字符串转义
 */
func (p *AttaApi) EscapeString(fieldValue string) (ret string) {
	ret = EscapeString(fieldValue)
	return
}

/**
 * 将字符串反转义
 */
func (p *AttaApi) UnescapeString(fieldValue string) (ret string) {
	return UnescapeString(fieldValue)
}

/**
 * 转义并拼接成一个字符串
 */
func (p *AttaApi) EscapeFields(fields []string) (ret string) {
	return EscapeFields(fields)
}

/**
 * 将一个字符串拆分，并对字段值进行反转义操作
 */
func (p *AttaApi) UnescapeFields(fieldValues string) []string {
	return UnescapeFields(fieldValues)
}

//释放函数：仅在需要频繁释放 attaapi 对象的地方使用，绝大多数场景不应该用
func (p *AttaApi) Release() {
	if p.conn != nil {
		p.conn.Close()
	}
	atomic.CompareAndSwapInt32(&p.reConGrState, attaReconGrStateCreated, attaReconGrStateToExit)
	atomic.CompareAndSwapInt32(&p.alarmGrState, attaAlarmGrStateCreated, attaAlarmGrStateToExit)
}
