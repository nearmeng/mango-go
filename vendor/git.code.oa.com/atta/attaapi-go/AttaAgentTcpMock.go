package attaapi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

const BODY_OFFSET int = 24
const SIGN_BODY_OFFSET int = 21
const IVERSION_OFFSET int = 4
const MSGTIME_OFFSET int = 7
const SIGN_TOTALLEN_OFFSET int = 0  // TCP签名响应总长度
const SIGN_IVERSION_OFFSET int = 4  // 版本号 默认0x02
const RESP_TOTALLEN_OFFSET int = 0  //	uint32_t iTotalLen;	//TCP上报响应总长度
const RESP_IVERSION_OFFSET int = 4  //	uint8_t iVersion;	//版本号，默认0x01
const RESP_IRESPCODE_OFFSET int = 5 //	uint8_t iRespCode;	//返回值，M_ATTA_REPORT_CODE_SUCCESS  0 //成功
const RESP_IMSGLEN_OFFSET int = 6   //	uint32_t iMsgLen;	//错误信息长度
const RESP_BODY_OFFSET int = 10

// const attaApiVerV1 byte = 0x01        // API协议版本标识：ATTAAPI V1.0，类型是消息
// const attaApiSignV1 byte = 0x02       // API协议版本标识：ATTAAPI V1.0，类型是签名
const SIGN_IMSGTIME_OFFSET int = 5   //	uint64_t lMsgTime;	//签名时间戳，毫秒数
const SIGN_ISIGNTIME_OFFSET int = 13 //uint64_t iSignTime;//签名时间，10进制，高低位置换，如12345改为54321

//AttaAgentTcpMock 服务器
type AttaAgentTcpMock struct {
	canStop        bool
	serivceListen  *net.TCPListener
	receiveChannel chan interface{}
	receiveBuffer  []byte
	signBuffer     []byte
	msgBuffer      []byte
}

func (this *AttaAgentTcpMock) init() int {
	// this.receiveBuffer = make([]byte,1024)

	// 初始化签名字节流
	{
		this.signBuffer = make([]byte, SIGN_BODY_OFFSET)
		bTotalLen := Int2Bytes(SIGN_BODY_OFFSET)
		copy(this.signBuffer[SIGN_TOTALLEN_OFFSET:SIGN_IVERSION_OFFSET], bTotalLen[:])
		this.signBuffer[SIGN_IVERSION_OFFSET] = attaApiSignV1
	}
	// 初始化成功信息字节流
	{
		bMsgSuccess := []byte("success")
		this.msgBuffer = make([]byte, RESP_BODY_OFFSET+len(bMsgSuccess))
		bTotalLen := Int2Bytes(len(this.msgBuffer))
		copy(this.msgBuffer[RESP_TOTALLEN_OFFSET:RESP_IVERSION_OFFSET], bTotalLen)
		this.msgBuffer[RESP_IVERSION_OFFSET] = attaApiVerV1
		this.msgBuffer[RESP_IRESPCODE_OFFSET] = 0 // ATTA REPORT CODE SUCCESS
		bMsgLen := Int2Bytes(len(bMsgSuccess))
		copy(this.msgBuffer[RESP_IMSGLEN_OFFSET:RESP_BODY_OFFSET], bMsgLen)
		copy(this.msgBuffer[RESP_BODY_OFFSET:RESP_BODY_OFFSET+len(bMsgSuccess)], bMsgSuccess)
	}

	this.canStop = false
	var tmp AttaAgentUdpMock
	var port int = tmp.findUsefulPort(attaAgentUDPHost, attaAgentUDPPorts)
	if port == 0 {
		return 0
	}
	addr, err := net.ResolveTCPAddr("tcp", attaAgentUDPHost+":"+strconv.Itoa(port))
	if err != nil {
		return 0
	}
	this.serivceListen, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return 0
	}
	return port
}

func (this *AttaAgentTcpMock) setCanstop(canStop bool) {
	this.canStop = canStop
	return
}

func (this *AttaAgentTcpMock) run() {
	for {
		conn, err := this.serivceListen.Accept()
		if err != nil {
			this.serivceListen.Close()
		}
		go handleRequests(conn, this.signBuffer)
		if this.canStop == true {
			break
		}
	}
	if this.serivceListen != nil {
		this.serivceListen.Close()
	}
}

func handleRequests(conn net.Conn, signBuffer []byte) {
	defer conn.Close()
	receiveData := make([]byte, 1024)
	cnt, err := conn.Read(receiveData)
	if err != nil {
		fmt.Println("handleRequests->conn.Read", err)
		return
	}
	fmt.Println("Expect length:" + strconv.Itoa(BODY_OFFSET) + " actual length:" + strconv.Itoa(cnt))

	iVersion := receiveData[IVERSION_OFFSET]
	switch iVersion {
	case attaApiSignV1:
		// 计算签名时间
		lMsgTime := int(binary.BigEndian.Uint64(receiveData[MSGTIME_OFFSET:]))
		signTime := 0
		current := lMsgTime
		var oneNum int
		for current > 0 {
			oneNum = current % 10
			current /= 10
			signTime = signTime*10 + oneNum
		}

		// 构建返回包
		var bMsgTime []byte = Long2Bytes(lMsgTime)
		var bSignTime []byte = Long2Bytes(signTime)
		copy(signBuffer[SIGN_IMSGTIME_OFFSET:SIGN_ISIGNTIME_OFFSET], bMsgTime)
		copy(signBuffer[SIGN_ISIGNTIME_OFFSET:SIGN_BODY_OFFSET], bSignTime)
		conn.Write(signBuffer[SIGN_TOTALLEN_OFFSET:SIGN_BODY_OFFSET])
	case attaApiVerV1:
		conn.Write(receiveData)
	default:
		fmt.Println("Unknown iversion:" + string(iVersion))
	}
}

//long 转 bytes
func Long2Bytes(n int) []byte {
	bytebuf := bytes.NewBuffer([]byte{})
	binary.Write(bytebuf, binary.BigEndian, int64(n))
	return bytebuf.Bytes()
}

//int 转 bytes
func Int2Bytes(n int) []byte {
	bytebuf := bytes.NewBuffer([]byte{})
	bytebuf.WriteByte(byte(n >> 24))
	bytebuf.WriteByte(byte(n >> 16))
	bytebuf.WriteByte(byte(n >> 8))
	bytebuf.WriteByte(byte(n))
	return bytebuf.Bytes()
}
