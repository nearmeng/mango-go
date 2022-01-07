package attaapi

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const MAX_ATTAMSG_LEN = 1024 * 64

//AttaAgentUdpMock 服务器
type AttaAgentUdpMock struct {
	canStop        bool
	serivceListen  *net.UDPConn
	receiveChannel chan interface{}
}

func (this *AttaAgentUdpMock) init() int {
	this.canStop = false
	var port int = this.findUsefulPort(attaAgentUDPHost, attaAgentUDPPorts)
	if port == 0 {
		return 0
	}
	addr, err := net.ResolveUDPAddr("udp", attaAgentUDPHost+":"+strconv.Itoa(port))
	if err != nil {
		return 0
	}
	this.serivceListen, err = net.ListenUDP("udp", addr)
	if err != nil {
		return 0
	}
	return port
}

func (this *AttaAgentUdpMock) setCanstop(canStop bool) {
	this.canStop = canStop
	return
}

func (this *AttaAgentUdpMock) findUsefulPort(_ip string, _ports string) int {
	var isOKPort bool = false
	var iPort int = 0
	portArray := strings.Split(_ports, ",")
	for _, port := range portArray {
		iPort, _ = strconv.Atoi(port)
		serverAddr := _ip + ":" + port
		_, err := net.DialTimeout("tcp", serverAddr, dialTimeOutMs*time.Millisecond)
		if err != nil && strings.Contains(err.Error(), "connection refused") {
			isOKPort = true
			break
		}
	}
	if isOKPort {
		return iPort
	}
	return 0
}

func (this *AttaAgentUdpMock) run() {
	receiveData := make([]byte, 1024)
	// defer this.setCanstop(true)
	for {
		n, remoteAddr, err := this.serivceListen.ReadFromUDP(receiveData)
		if err != nil {
			this.serivceListen.Close()
		}
		fmt.Println(n, remoteAddr, err)
		this.receiveChannel <- n
		if this.canStop == true {
			break
		}
	}
	if this.serivceListen != nil {
		this.serivceListen.Close()
	}
}
