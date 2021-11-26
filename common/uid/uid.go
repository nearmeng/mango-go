package uid

import (
	"sync"
	"time"
)

const (
	_baseTimeStamp   = 1629941248
	_funcIDUIDOffset = 56
	_insIDUIDOffset  = 45
	_timeBitNum      = 29
	_seqBitNum       = 16
)

type uidGeneratorParam struct {
	mu        sync.Mutex
	lastStamp int64
	prefixUID uint64
	sequence  int64
	seqBits   int8
	timeBits  int8
}

type uidGenerator struct {
	genParam uidGeneratorParam
}

var uidGeneratorIns *uidGenerator

// InitUIDGenerator 初始化uid生成器.
func InitUIDGenerator(funcID uint8, insID uint32) {
	uidGeneratorIns = &uidGenerator{
		genParam: uidGeneratorParam{
			lastStamp: 0,
			// 8个字节funcid(0-255), 11个字节insid:(0-2047), 29字节时间戳(可支持17年左右), 16字节序号:(0-65535),也就是同1s单个进程支持65535个不重复的id
			prefixUID: (uint64(funcID) << _funcIDUIDOffset) + (uint64(insID) << _insIDUIDOffset),
			sequence:  0,
			seqBits:   _seqBitNum,
			timeBits:  _timeBitNum,
		},
	}
}

// GenerateUID 生成uid.
func GenerateUID() uint64 {
	if uidGeneratorIns == nil {
		return 0
	}
	uidGeneratorIns.genParam.mu.Lock()
	defer uidGeneratorIns.genParam.mu.Unlock()
	return uidGeneratorIns.doGenerate(&uidGeneratorIns.genParam)
}

func (g *uidGenerator) doGenerate(genParam *uidGeneratorParam) uint64 {
	nowTime := time.Now().Unix()
	if nowTime < _baseTimeStamp {
		return 0
	}

	if genParam.lastStamp != nowTime {
		genParam.lastStamp = nowTime
		genParam.sequence = 0
	}

	temp := nowTime - _baseTimeStamp
	result := genParam.prefixUID + (uint64(temp) << genParam.seqBits)

	if genParam.sequence > (1<<genParam.seqBits)-1 {
		return 0
	}

	result += uint64(genParam.sequence)

	genParam.sequence++

	return result
}
