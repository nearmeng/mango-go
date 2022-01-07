package attaapi

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const alarmIpList = "9.146.136.245;9.146.137.234;9.146.138.165;9.146.137.172"
const urlModel = "http://%s:7788/agentnotfound?iplist=%s&attaid=%s&version=%s&rundir=%s&md5=%s"
const md5Format = "attaid=%s&iplist=%s&rundir=%s&version=%s"
const alarmTimeout = 5
const alarmLocalIpSize = 3

func (p *AttaApi) sendAlarm() {
	if atomic.CompareAndSwapInt32(&p.alarmGrState, attaAlarmGrStateNull, attaAlarmGrStateCreated) {
		go p.doSendAlarm()
	}
}

func (p *AttaApi) doSendAlarm() {
	tickTimer := time.NewTicker(time.Second)
	defer tickTimer.Stop()
	ipList := strings.Split(alarmIpList, ";")
	ipIndex := time.Now().Unix() % int64(len(ipList))
	for {
		select {
		case <-tickTimer.C:
			{
				if atomic.CompareAndSwapInt32(&p.alarmGrState, attaAlarmGrStateToExit, attaAlarmGrStateNull) {
					return
				}
				if p.conState == attaConStateOK {
					if atomic.CompareAndSwapInt32(&p.alarmGrState, attaAlarmGrStateCreated, attaAlarmGrStateNull) {
						//fmt.Printf("\nsendAlarm close")
						return
					}
				}
				if p.conState != attaConStateOK && time.Since(p.alarmTime) > alarmInterval*time.Second {
					client := http.Client{Timeout: alarmTimeout * time.Second}
					ipIndex++
					temp := ipIndex % int64(len(ipList))
					resp, err := client.Get(p.getAlarmUrl(ipList[temp]))
					if err == nil {
						resp.Body.Close()
					}
					p.alarmTime = time.Now()
				}
			}
		}
	}
}

func (p *AttaApi) getAlarmUrl(ip string) string {
	localIpList := p.getLocalIplist(alarmLocalIpSize)
	ipList := strings.Join(localIpList, ",")
	temp := fmt.Sprintf(md5Format, p.alarmAttaId, ipList, os.Args[0], attaApiVerAlarm)
	data := []byte(temp)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has)
	url := fmt.Sprintf(urlModel, ip, ipList, p.alarmAttaId, attaApiVerAlarm, os.Args[0], md5str)
	return url
}
