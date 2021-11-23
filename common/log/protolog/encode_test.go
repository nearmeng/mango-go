// package prototlog is a configurable text format marshaler for tlog.
package protolog

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"git.woa.com/bingo/bingo/db/test"
	"github.com/stretchr/testify/assert"
)

func TestMarshalOptions_Marshal(t *testing.T) {
	m := &test.TBTest{
		Uid: 123,
		BaseInfo: &test.STAcntBaseInfo{
			Name: "xx",
		},
		Rank1V1: &test.STRank1V1{
			Elo: 123411,
		},
		IncTest1:     -1,
		IncTest2:     0xfffffffff,
		FloatTest1:   math.Pi,
		FloatTest2:   -math.Pi,
		BoolTest1:    true,
		TestStr:      "TestStr",
		IntArray:     []int32{1, 2},
		IntStringMap: map[int64]string{1: "IntStringMap1"},
	}

	rs := MarshalOptions{}.Format(m)
	_, _ = fmt.Println("retstr=", rs)
	assert.True(t, strings.HasPrefix(rs, "message"))
	assert.True(t, strings.Contains(rs, "uid=123"))
	assert.True(t, strings.Contains(rs, "Rank1V1.Elo=123411"))
	assert.True(t, strings.Contains(rs, "boolTest1=true"))
	assert.True(t, strings.Contains(rs, "IntArray.0=1"))
	assert.True(t, strings.Contains(rs, "IntArray.1=2"))
	assert.True(t, strings.Contains(rs, "IntStringMap.1=IntStringMap1"))
	assert.True(t, strings.Contains(rs, "FloatTest1=3.141592"))
	assert.True(t, strings.Contains(rs, "FloatTest2=-3.141592"))
	assert.True(t, strings.Contains(rs, "FloatTest2=-3.141592"))
}
