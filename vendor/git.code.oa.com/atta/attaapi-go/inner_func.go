package attaapi

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	maxMsgLen      int = 63 * 1024  // 用户上报数据长度限制(UDP)
	maxMsgLen4Tcp  int = 500 * 1024 // 用户上报数据长度限制(TCP)
	maxPackLen     int = 64 * 1024  // 协议总长度限制(UDP)
	maxPackLen4Tcp int = 510 * 1024 // 协议总长度限制(TCP)
	maxMsgNum      int = 100        // 批量上报条数限制
)

// 包体偏移
const bodyOffset int = 24

//	接收uint32_t iTotalLen;	//接收总长度
const mrTotalLenOffset int = 0

//	接收uint8_t iVersion;	//版本号，默认0x01
const mrIVersionOffset int = 4

//	接收uint8_t iRespCode;	//返回值，M_ATTA_REPORT_CODE_SUCCESS  0 //成功
const mrRespCodeOffset int = 5

//	接收包体偏移
const mrMessageOffset int = 10

//	签名响应uint32_t iTotalLen;	//总长度
const srTotalLenOffset int = 0

//	签名响应uint8_t iVersion;	//版本号，默认0x02
const srIVersionOffset int = 4

//	签名响应uint64_t lMsgTime;	//签名时间戳，毫秒数
const srMsgTimeOffset int = 5

//	签名响应uint64_t iSignTime;	//签名时间，10进制，高低位置换，如12345改为54321
const srSignTimeOffset int = 13

//	签名响应包体偏移
const srMessageOffset int = 21

const attaApiSchemaSimple byte = 0x10 // API内容格式标识：简单协议（分隔符、透传数据）
const attaApiSchemaKV byte = 0x20     // API内容格式标识：Key-value
const attaApiVerV1 byte = 0x01        // API协议版本标识：ATTAAPI V1.0，类型是消息
const attaApiSignV1 byte = 0x02       // API协议版本标识：ATTAAPI V1.0，类型是签名

const attaApiVerAlarm = "go1.1" // 告警时上报安排api版本号

const sourceAtta = 3
const sourceMAtta = 5 //AttaId批量上报包

const socketTypeUnix = 0 //unix域套接字
const socketTypeTcp = 1  //TCP
const socketTypeUdp = 2  //UDP

const attaDcClientIP1 = "__clientip="

const dialTimeOutMs = 10
const reInitInterval = 3
const alarmInterval = 600

const attaReconGrStateNull = 0
const attaReconGrStateCreated = 1
const attaReconGrStateToExit = 2

const attaAlarmGrStateNull = 0
const attaAlarmGrStateCreated = 1
const attaAlarmGrStateToExit = 2

const attaConStateNotInit = 0
const attaConStateOK = 1
const attaConStateToRecon = 2

const attaInitStateNotInit = 0
const attaInitStateInitSuc = 1

var defaultSeparator byte = '|'

const lenOfDataLen = 4
const lenOfBatchNum = 2
const lenOfTag = 1
const lenOfValueLen = 1
const tag4SdkVersion = 0x00

const reconTimes = 5
const reconMs = 10
const defaultRWTimeout = 10
const rwTimeoutMin = 1
const rwTimeoutMax = 10000

func (p *AttaApi) init(nType int) int {
	ret := AttaReportCodeUnknownSocketType
	if nType == socketTypeUdp {
		ret = p.doInitUDP()
	} else if nType == socketTypeTcp {
		ret = p.doInitTCP()
	} else if nType == socketTypeUnix {
		ret = p.doInitUnix()
	}
	if ret == AttaReportCodeSuccess {
		atomic.CompareAndSwapInt32(&p.initState, attaInitStateNotInit, attaInitStateInitSuc)
		atomic.CompareAndSwapInt32(&p.conState, attaConStateNotInit, attaConStateOK)
	} else {
		p.sendAlarm()
	}
	return ret
}

func (p *AttaApi) doInitUDP() int {
	p.m_type = socketTypeUdp
	p.m_ip = attaAgentUDPHost
	usefulPort := p.findUsefulPort(p.m_ip, attaAgentUDPPorts)
	if usefulPort == 0 {
		return AttaReportCodeNoUsablePort
	} else {
		p.m_port = strconv.Itoa(usefulPort)
	}
	return p.dialAgent("udp", p.m_ip, p.m_port)
}

func (p *AttaApi) doInitTCP() int {
	p.m_type = socketTypeTcp
	p.m_ip = attaAgentUDPHost
	p.rwTimeout = defaultRWTimeout
	usefulPort := p.findUsefulPort(p.m_ip, attaAgentUDPPorts)
	if usefulPort == 0 {
		return AttaReportCodeNoUsablePort
	} else {
		p.m_port = strconv.Itoa(usefulPort)
	}
	return p.dialAgent("tcp", p.m_ip, p.m_port)
}

func (p *AttaApi) doInitUnix() int {
	p.m_type = socketTypeUnix
	p.m_path = attaAgentUnixPath
	addr, err := net.ResolveUnixAddr("unix", p.m_path)
	if err != nil {
		return AttaReportCodeCreateSocketFailed
	}
	//拔号
	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return AttaReportCodeConnetSocketFailed
	}
	p.conn = conn
	return AttaReportCodeSuccess
}

func (p *AttaApi) sendBinary(sAttaid string, sToken string, bData [][]byte, source int) int {
	if bData == nil || len(bData) <= 0 {
		return AttaReportCodeInvalidMsg
	}

	dataLen := 0
	msgsLen := make([]int, len(bData))
	for msgIndex, msg := range bData {
		msgsLen[msgIndex] = len(msg)
		dataLen += msgsLen[msgIndex]
	}
	// 校验数据内容长度
	rec := p.checkDataLen(dataLen)
	if rec != AttaReportCodeSuccess {
		return rec
	}
	if source == sourceMAtta {
		dataLen += lenOfDataLen * len(bData)
	}

	buf := GetBuf()
	defer PutBuf(buf)
	totalLen := p.fillReqHead(buf, sAttaid, sToken, dataLen, source, len(msgsLen))
	// 校验总长度
	rec = p.checkPacLen(int(totalLen))
	if rec != AttaReportCodeSuccess {
		return rec
	}
	//内容
	for msgIndex, msg := range bData {
		if source == sourceMAtta {
			PutUint32InBigEndian(buf, uint32(msgsLen[msgIndex]))
		}
		buf.Write(msg)
	}

	ret := p.send2Conn(sAttaid, buf.Bytes(), totalLen)
	return ret
}

func (p *AttaApi) sendString(sAttaid string, sToken string, sData []string, source int) int {
	if len(sData) <= 0 {
		return AttaReportCodeInvalidMsg
	}

	dataLen := 0
	msgsLen := make([]int, len(sData))
	for msgIndex, msg := range sData {
		msgsLen[msgIndex] = len(msg)
		dataLen += msgsLen[msgIndex]
	}
	// 校验数据内容长度
	rec := p.checkDataLen(dataLen)
	if rec != AttaReportCodeSuccess {
		return rec
	}
	if source == sourceMAtta {
		dataLen += lenOfDataLen * len(sData)
	}

	buf := GetBuf()
	defer PutBuf(buf)
	totalLen := p.fillReqHead(buf, sAttaid, sToken, dataLen, source, len(msgsLen))
	// 校验总长度
	rec = p.checkPacLen(int(totalLen))
	if rec != AttaReportCodeSuccess {
		return rec
	}
	//内容
	for msgIndex, msg := range sData {
		if source == sourceMAtta {
			PutUint32InBigEndian(buf, uint32(msgsLen[msgIndex]))
		}
		buf.WriteString(msg)
	}

	ret := p.send2Conn(sAttaid, buf.Bytes(), totalLen)
	return ret
}

func (p *AttaApi) sendStringFields(sAttaid string, sToken string, _fields [][]string, source int) int {
	if _fields == nil || len(_fields) == 0 {
		return AttaReportCodeInvalidMsg
	}

	dataLen := 0
	msgsLen := make([]int, len(_fields))
	for msgIndex, msg := range _fields {
		msgsLen[msgIndex] = len(msg) - 1
		for _, field := range msg {
			msgsLen[msgIndex] += len(field)
		}
		dataLen += msgsLen[msgIndex]
	}
	// 校验数据内容长度
	rec := p.checkDataLen(dataLen)
	if rec != AttaReportCodeSuccess {
		return rec
	}
	if source == sourceMAtta {
		dataLen += lenOfDataLen * len(_fields)
	}

	buf := GetBuf()
	defer PutBuf(buf)
	totalLen := p.fillReqHead(buf, sAttaid, sToken, dataLen, source, len(msgsLen))
	// 校验总长度
	rec = p.checkPacLen(int(totalLen))
	if rec != AttaReportCodeSuccess {
		return rec
	}
	for msgIndex, msg := range _fields {
		if source == sourceMAtta {
			PutUint32InBigEndian(buf, uint32(msgsLen[msgIndex]))
		}
		for fieldIndex, field := range msg {
			buf.WriteString(field)
			if fieldIndex < len(msg)-1 {
				buf.WriteByte(defaultSeparator)
			}
		}
	}

	ret := p.send2Conn(sAttaid, buf.Bytes(), totalLen)
	return ret
}

func (p *AttaApi) sendStringFieldsAutoEscape(sAttaid string, sToken string, _fields [][]string, source int) int {
	if _fields == nil || len(_fields) == 0 {
		return AttaReportCodeInvalidMsg
	}

	dataLen := 0
	fieldsBuf := make([][]*bytes.Buffer, len(_fields))
	msgsLen := make([]int, len(_fields))
	var escBufs []*bytes.Buffer
	for msgIndex, msg := range _fields {
		msgsLen[msgIndex] = len(msg) - 1
		for fieldIndex, field := range msg {
			fieldBuf := GetBuf()
			fieldsBuf[msgIndex] = append(fieldsBuf[msgIndex], fieldBuf)
			escBufs = append(escBufs, fieldBuf)
			msgsLen[msgIndex] += escapeString(fieldsBuf[msgIndex][fieldIndex], field)
		}
		dataLen += msgsLen[msgIndex]
	}
	defer func() {
		for _, escBuf := range escBufs {
			PutBuf(escBuf)
		}
	}()
	// 校验数据内容长度
	rec := p.checkDataLen(dataLen)
	if rec != AttaReportCodeSuccess {
		return rec
	}
	if source == sourceMAtta {
		dataLen += lenOfDataLen * len(_fields)
	}

	buf := GetBuf()
	defer PutBuf(buf)
	totalLen := p.fillReqHead(buf, sAttaid, sToken, dataLen, source, len(msgsLen))
	// 校验总长度
	rec = p.checkPacLen(int(totalLen))
	if rec != AttaReportCodeSuccess {
		return rec
	}
	for msgIndex, msgBuf := range fieldsBuf {
		if source == sourceMAtta {
			PutUint32InBigEndian(buf, uint32(msgsLen[msgIndex]))
		}
		for fieldIndex, fieldBuf := range msgBuf {
			buf.Write(fieldBuf.Bytes())
			if fieldIndex < len(msgBuf)-1 {
				buf.WriteByte(defaultSeparator)
			}
		}
	}
	ret := p.send2Conn(sAttaid, buf.Bytes(), totalLen)
	return ret
}

// 校验数据内容长度
func (p *AttaApi) checkDataLen(dataLen int) int {
	if dataLen <= 0 {
		return AttaReportCodeInvalidMsg
	}

	maxLen := maxMsgLen
	if p.m_type == socketTypeTcp {
		maxLen = maxMsgLen4Tcp
	}
	if dataLen > maxLen {
		return AttaReportCodeOverLimit
	}

	return AttaReportCodeSuccess
}

// 校验包长度
func (p *AttaApi) checkPacLen(dataLen int) int {
	if dataLen <= 0 {
		return AttaReportCodeInvalidMsg
	}

	maxLen := maxPackLen
	if p.m_type == socketTypeTcp {
		maxLen = maxPackLen4Tcp
	}
	if dataLen > maxLen {
		return AttaReportCodeOverLimit
	}

	return AttaReportCodeSuccess
}

// 校验批量上报数据条数
func (p *AttaApi) checkDataNum(dataNum int) int {
	if dataNum <= 0 {
		return AttaReportCodeInvalidMsg
	}
	if dataNum > maxMsgNum {
		return AttaReportCodeOverBatchNum
	}
	return AttaReportCodeSuccess
}

func (p *AttaApi) fillReqHead(buf *bytes.Buffer, sAttaid string, sToken string,
	dataLen int, source int, batchSize int) uint32 {
	token := sToken
	if len(sToken) == 0 {
		token = "0"
	}
	attaIdLen := len(sAttaid)
	tokenLen := len(token)
	ipLen := len(p.m_localip)

	iResLen := uint16(lenOfBatchNum + lenOfTag + lenOfValueLen + len(attaApiVerAlarm))
	totalLen := uint32(bodyOffset + (int)(iResLen) + attaIdLen + tokenLen + ipLen + dataLen)
	//		uint32_t iTotalLen;	//总长度
	PutUint32InBigEndian(buf, totalLen)
	//		uint8_t iVersion;	//版本号，默认0x01
	buf.WriteByte(byte(attaApiVerV1))
	//		uint8_t iSource;		//消息来源系统，如M_ATTA_SOURCE_AT
	buf.WriteByte(byte(source))
	//		uint8_t cVersion;	//Boss来源专用，默认0x10：0x10分隔符,0x20-URL的key value协议,0x40-json
	buf.WriteByte(byte(attaApiSchemaSimple))
	//		uint64_t lMsgTime;	//消息生成时间
	msgTime := time.Now().UnixNano() / 1000000
	PutUint64InBigEndian(buf, uint64(msgTime))
	//		uint16_t iResLen;	//预留信息长度，默认为0
	PutUint16InBigEndian(buf, iResLen)
	//		uint8_t iAttaIdLen;	//AttaId长度
	buf.WriteByte(byte(attaIdLen))
	//		uint8_t iPwdLen;	//密钥长度
	buf.WriteByte(byte(tokenLen))
	//		uint8_t iIpLen;		//消息上报IP长度
	buf.WriteByte(byte(ipLen))
	//		uint32_t iValueLen;	//内容长度
	PutUint32InBigEndian(buf, uint32(dataLen))

	PutUint16InBigEndian(buf, uint16(batchSize))

	buf.WriteByte(byte(tag4SdkVersion))

	buf.WriteByte(byte(len(attaApiVerAlarm)))

	buf.WriteString(attaApiVerAlarm)
	//attaid
	buf.WriteString(sAttaid)
	//密钥
	buf.WriteString(token)
	//ip
	buf.WriteString(p.m_localip)

	return totalLen
}

func (p *AttaApi) send2Conn(attaId string, sendData []byte, totalLen uint32) int {
	if attaInitStateNotInit == p.initState {
		p.alarmAttaId = attaId
		return AttaReportCodeNotInit
	}
	var ret int
	if p.m_type == socketTypeUdp {
		ret = p.sendByUdp(attaId, sendData, totalLen)
	} else if p.m_type == socketTypeTcp {
		ret = p.sendByTcp(attaId, sendData, totalLen)
	} else if p.m_type == socketTypeUnix {
		ret = p.sendByUnixSocket(attaId, sendData, totalLen)
	} else {
		ret = AttaReportCodeUnknownSocketType
	}
	return ret
}

func (p *AttaApi) sendByUdp(attaId string, sendData []byte, totalLen uint32) int {
	if p.conn == nil {
		p.reInit(attaId)
		return AttaReportCodeNotInit
	}
	nLen, err := p.conn.Write(sendData)
	if err != nil || nLen != int(totalLen) {
		p.reInit(attaId)
		return AttaReportCodeNetFailed
	}
	return AttaReportCodeSuccess
}

func (p *AttaApi) reInit(attaId string) {
	p.alarmAttaId = attaId
	if atomic.CompareAndSwapInt32(&p.reConGrState, attaReconGrStateNull, attaReconGrStateCreated) {
		go p.doReInit()
	}
	atomic.CompareAndSwapInt32(&p.conState, attaConStateOK, attaConStateToRecon)
	p.sendAlarm()
}

func (p *AttaApi) doReInit() {
	tickTimer := time.NewTicker(time.Second)
	defer tickTimer.Stop()
	for {
		select {
		case <-tickTimer.C:
			{
				if atomic.CompareAndSwapInt32(&p.reConGrState, attaReconGrStateToExit, attaReconGrStateNull) {
					return
				}
				ret := AttaReportCodeConnetSocketFailed
				if p.conState == attaConStateToRecon && time.Since(p.reConTime) > reInitInterval*time.Second {
					if p.m_type == socketTypeUdp {
						ret = p.InitUDP()
					} else if p.m_type == socketTypeTcp {
						ret = p.InitTCP()
					} else if p.m_type == socketTypeUnix {
						ret = p.InitUnix()
					}
					p.reConTime = time.Now()
					if ret == AttaReportCodeSuccess {
						atomic.CompareAndSwapInt32(&p.conState, attaConStateToRecon, attaConStateOK)
						if atomic.CompareAndSwapInt32(&p.reConGrState, attaReconGrStateCreated, attaReconGrStateNull) {
							return
						}
					}
				}
			}
		}
	}
}

func (p *AttaApi) sendByTcp(attaId string, sendData []byte, totalLen uint32) int {
	if p.conn == nil {
		p.reInit(attaId)
		return AttaReportCodeNotInit
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	wroteLen := 0
	p.conn.SetDeadline(time.Now().Add(time.Duration(p.rwTimeout) * time.Millisecond))
	for {
		nLen, err := p.conn.Write(sendData[wroteLen:totalLen])
		if err != nil {
			p.reInit(attaId)
			return AttaReportCodeTCPWriteFailed
		}
		wroteLen += nLen
		if wroteLen >= int(totalLen) {
			break
		}
	}
	// 接收
	readLen := 0
	for {
		nLen, err := p.conn.Read(sendData[readLen:])
		if err != nil {
			p.reInit(attaId)
			return AttaReportCodeTCPReadFailed
		}
		readLen += nLen
		if readLen >= mrMessageOffset {
			mrTotalLen := binary.BigEndian.Uint32(sendData[mrTotalLenOffset:mrIVersionOffset])
			if readLen >= int(mrTotalLen) {
				break
			}
		}
	}
	mrRespCode := sendData[mrRespCodeOffset]
	if int(mrRespCode) != AttaReportCodeSuccess {
		if int(mrRespCode) != AgentReturnCodeBusy {
			p.reInit(attaId)
		}
		return AttaReportCodeAgentBusy
	}
	return AttaReportCodeSuccess
}

func (p *AttaApi) sendByUnixSocket(attaId string, sendData []byte, totalLen uint32) int {
	if p.conn == nil {
		p.reInit(attaId)
		return AttaReportCodeNotInit
	}
	nLen, err := p.conn.Write(sendData)
	if err != nil || nLen != int(totalLen) {
		//fmt.Println("Write Unix error:", err, ",send size:", nLen, ",expect:", totalLen)
		p.reInit(attaId)
		return AttaReportCodeNetFailed
	}
	return AttaReportCodeSuccess
}

//转义函数
func EscapeString(fieldValue string) (ret string) {
	buf := GetBuf()
	escapeString(buf, fieldValue)
	ret = buf.String()
	PutBuf(buf)
	return
}

func escapeString(buf *bytes.Buffer, fieldValue string) int {
	var separator byte = '|'
	fieldLength := len(fieldValue)
	if fieldLength <= 0 {
		return 0
	}
	fieldLength = len(fieldValue)
	start := 0
	for i := 0; i < fieldLength; i++ {
		switch fieldValue[i] {
		case 0:
			buf.WriteString(fieldValue[start:i])
			buf.WriteByte('\\')
			buf.WriteByte('0')
			start = i + 1
			break
		case '\n':
			buf.WriteString(fieldValue[start:i])
			buf.WriteByte('\\')
			buf.WriteByte('n')
			start = i + 1
			break
		case '\r':
			buf.WriteString(fieldValue[start:i])
			buf.WriteByte('\\')
			buf.WriteByte('r')
			start = i + 1
			break
		case '\\':
			buf.WriteString(fieldValue[start:i])
			buf.WriteByte('\\')
			buf.WriteByte('\\')
			start = i + 1
			break
		default:
			if fieldValue[i] == separator {
				buf.WriteString(fieldValue[start:i])
				buf.WriteByte('\\')
				buf.WriteByte(separator)
				start = i + 1
			}
			break
		}
	}
	if start != fieldLength {
		buf.WriteString(fieldValue[start:])
	}
	return len(buf.Bytes())
}

//反转义函数
func UnescapeString(fieldValue string) (ret string) {
	var separator byte = '|'
	fieldLength := len(fieldValue)
	if fieldLength <= 0 {
		return ""
	}
	buf := GetBuf()
	bData := []byte(fieldValue)
	fieldLength = len(bData)

	var i int = 0
	for i = 0; i < fieldLength-1; i++ {
		value := bData[i]
		switch value {
		case '\\':
			nextValue := bData[i+1]
			switch nextValue {
			case '0':
				buf.WriteByte(0)
				i++
				break
			case 'n':
				buf.WriteByte('\n')
				i++
				break
			case 'r':
				buf.WriteByte('\r')
				i++
				break
			case '\\':
				buf.WriteByte('\\')
				i++
				break
			default:
				if nextValue == separator {
					buf.WriteByte(separator)
					i++
				} else {
					buf.WriteByte(value)
				}
				break
			}
			break
		default:
			buf.WriteByte(value)
			break
		}
	}

	if i == fieldLength-1 {
		value := bData[i]
		buf.WriteByte(value)
	}
	ret = buf.String()
	PutBuf(buf)
	return
}

/**
 * 转义并拼接成一个字符串
 */
func EscapeFields(fields []string) (ret string) {
	if len(fields) <= 0 {
		return ""
	}
	var separator byte = '|'
	buf := GetBuf()
	fieldLength := len(fields)
	for i := 0; i < fieldLength; i++ {
		fmtField := EscapeString(fields[i])
		buf.WriteString(fmtField)
		buf.WriteByte(separator)
	}
	ret = buf.String()
	PutBuf(buf)
	return
}

/**
 * 将一个字符串拆分，并对字段值进行反转义操作
 */
func UnescapeFields(fieldValues string) []string {
	if len(fieldValues) <= 0 {
		return nil
	}
	fieldList := list.New()
	var separator byte = '|'

	bData := []byte(fieldValues)
	fieldLength := len(bData)
	buf := GetBuf()
	var i = 0
	for i = 0; i < fieldLength-1; i++ {
		value := bData[i]
		switch value {
		case '\\':
			nextValue := bData[i+1]
			switch nextValue {
			case '0':
				buf.WriteByte(0)
				i++
				break
			case 'n':
				buf.WriteByte('\n')
				i++
				break
			case 'r':
				buf.WriteByte('\r')
				i++
				break
			case '\\':
				buf.WriteByte('\\')
				i++
				break
			default:
				if nextValue == separator {
					buf.WriteByte(separator)
					i++
				} else {
					buf.WriteByte(value)
				}
				break
			}
			break
		case '|':
			fieldValue := buf.String()
			fieldList.PushBack(fieldValue)
			buf.Reset()
			break
		default:
			buf.WriteByte(value)
			break
		}
	}

	if i == fieldLength-1 {
		value := bData[i]
		if value == separator {
			fieldValue := buf.String()
			fieldList.PushBack(fieldValue)
			fieldList.PushBack("")
		} else {
			buf.WriteByte(value)
			fieldValue := buf.String()
			fieldList.PushBack(fieldValue)
		}
	}
	PutBuf(buf)
	fieldsNum := fieldList.Len()
	fieldArray := make([]string, fieldsNum)
	i = 0
	for e := fieldList.Front(); e != nil; e = e.Next() {
		fieldArray[i] = e.Value.(string)
		i++
	}
	return fieldArray
}

func (p *AttaApi) dialAgent(netWork string, _ip string, _port string) int {
	p.m_ip = _ip
	p.m_port = _port
	serverAddr := p.m_ip + ":" + p.m_port
	var conn net.Conn
	var err error
	dialSuc := false
	for i := 0; i < reconTimes; i++ {
		conn, err = net.DialTimeout(netWork, serverAddr, dialTimeOutMs*time.Millisecond)
		if err == nil {
			dialSuc = true
			break
		}
		time.Sleep(time.Millisecond * reconMs)
	}
	if !dialSuc {
		return AttaReportCodeNoUsablePort
	} else {
		p.conn = conn
	}

	// 获取本机IP
	p.m_localip = "127.0.0.1"
	localIps := p.getLocalIplist(1)
	if len(localIps) == 0 {
		return AttaReportCodeConnetSocketFailed
	}
	p.m_localip = localIps[0]
	return AttaReportCodeSuccess
}

func (p *AttaApi) getLocalIplist(limit int) []string {
	var ret []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ret
	}
	count := 0
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				if !isInnerIp(ipnet.IP.String()) {
					continue
				}
			} else if ipnet.IP.To16() != nil {

			} else {
				continue
			}
			ret = append(ret, ipnet.IP.String())
			count++
			if count >= limit {
				break
			}
		}
	}
	if len(ret) == 0 {
		ret = append(ret, "127.0.0.1")
	}
	return ret
}

func isInnerIp(ipv4 string) bool {
	temp := strings.Split(ipv4, ".")
	firstNum, _ := strconv.Atoi(temp[0])
	if firstNum == 100 || firstNum == 172 || firstNum == 192 {
		return true
	}
	if firstNum >= 1 && firstNum <= 15 {
		return true
	}
	return false

}

//写 uint 16
func PutUint16InBigEndian(buf *bytes.Buffer, v uint16) {
	buf.WriteByte(byte(v >> 8))
	buf.WriteByte(byte(v))
}

//写 uint 32
func PutUint32InBigEndian(buf *bytes.Buffer, v uint32) {
	buf.WriteByte(byte(v >> 24))
	buf.WriteByte(byte(v >> 16))
	buf.WriteByte(byte(v >> 8))
	buf.WriteByte(byte(v))
}

//写 uint 64
func PutUint64InBigEndian(buf *bytes.Buffer, v uint64) {
	buf.WriteByte(byte(v >> 56))
	buf.WriteByte(byte(v >> 48))
	buf.WriteByte(byte(v >> 40))
	buf.WriteByte(byte(v >> 32))
	buf.WriteByte(byte(v >> 24))
	buf.WriteByte(byte(v >> 16))
	buf.WriteByte(byte(v >> 8))
	buf.WriteByte(byte(v))
}

//获取签名时间
func getTime4Sign(msgTime uint64) uint64 {
	var signTime uint64 = 0
	current := msgTime
	for {
		oneNum := current % 10
		current /= 10
		signTime = signTime*10 + oneNum
		if current <= 0 {
			break
		}
	}
	return signTime
}

func (p *AttaApi) findUsefulPort(_ip string, _ports string) int {
	if len(_ports) <= 0 {
		return attaAgentUDPPort
	}
	// 当前时间
	msgTime := uint64(time.Now().UnixNano() / 1000000)
	signTime := getTime4Sign(msgTime)
	receiveData := make([]byte, srMessageOffset)
	buf := GetBuf()
	totalLen := uint32(bodyOffset)
	PutUint32InBigEndian(buf, totalLen) //uint32_t iTotalLen;	//总长度
	buf.WriteByte(byte(attaApiSignV1))  //uint8_t iVersion;	    //版本号，默认0x02
	buf.WriteByte(0)                    //uint8_t iSource;          //无效，默认为0
	buf.WriteByte(0)                    //uint8_t cVersion;         //无效，默认为0
	PutUint64InBigEndian(buf, msgTime)  //uint64_t lMsgTime;	    //消息生成时间
	PutUint16InBigEndian(buf, 0)        //uint16_t iResLen;	        //无效，默认为0
	buf.WriteByte(0)                    //uint8_t iAttaIdLen;	    //无效，默认为0
	buf.WriteByte(0)                    //uint8_t iPwdLen;	        //无效，默认为0
	buf.WriteByte(0)                    //uint8_t iIpLen;		    //无效，默认为0
	PutUint32InBigEndian(buf, 0)        //uint32_t iValueLen;	    //无效，默认为0

	// 分拆ports字符串
	portArray := strings.Split(_ports, ",")
	for _, port := range portArray {
		serverAddr := _ip + ":" + port
		var conn net.Conn
		var err error
		dialSuc := false
		for i := 0; i < reconTimes; i++ {
			conn, err = net.DialTimeout("tcp", serverAddr, dialTimeOutMs*time.Millisecond)
			if err == nil {
				dialSuc = true
				break
			}
			time.Sleep(time.Millisecond * reconMs)
		}
		if !dialSuc {
			continue
		}
		// 发送
		wroteLen, err := conn.Write(buf.Bytes())
		if err != nil || wroteLen < int(totalLen) {
			conn.Close()
			continue
		}
		// 接收
		nLen, err := conn.Read(receiveData)
		if err != nil || nLen < srMessageOffset {
			conn.Close()
			continue
		}
		// 解析
		// uint32_t iTotalLen;	//总长度
		srTotalLen := binary.BigEndian.Uint32(receiveData[srTotalLenOffset:srIVersionOffset])
		// uint8_t iVersion;	//版本号，默认0x02
		srVersion := receiveData[srIVersionOffset]
		// uint64_t lMsgTime;	//签名时间戳，毫秒数
		srMsgTime := binary.BigEndian.Uint64(receiveData[srMsgTimeOffset:srSignTimeOffset])
		// uint64_t iSignTime;	//签名时间，10进制，高低位置换，如12345改为54321
		srSignTime := binary.BigEndian.Uint64(receiveData[srSignTimeOffset:srMessageOffset])
		if nLen != int(srTotalLen) {
			conn.Close()
			continue
		}

		// 判断签名
		if srVersion == attaApiSignV1 && srMsgTime == msgTime && srSignTime == signTime {
			iPort, err := strconv.Atoi(port)
			if err != nil {
				conn.Close()
				continue
			}
			conn.Close()
			PutBuf(buf)
			return iPort
		}
	}
	PutBuf(buf)
	return 0
}
