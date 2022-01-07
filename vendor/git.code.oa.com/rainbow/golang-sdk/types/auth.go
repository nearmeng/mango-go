package types

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"math"
	"math/rand"
	"time"
)

// SignatureParams 签名对象参数
type SignatureParams struct {
	SgnType   string `json:"rainbow_sgn_type"`   // 签名类型，现固定apisign
	Version   string `json:"rainbow_version"`    // 版本号，2020
	AppID     string `json:"rainbow_app_id"`     // 填
	UserID    string `json:"rainbow_user_id"`    // 填
	Timestamp string `json:"rainbow_timestamp"`  // 自动填充
	Nonce     string `json:"rainbow_nonce"`      // 自动填充
	SgnMethod string `json:"rainbow_sgn_method"` // 签名方式: sha256或sha1，默认sha1
	SgnBody   string `json:"rainbow_sgn_body"`   // 包体签名字符串，目前不填充
	Signature string `json:"rainbow_signature"`  // 签名串
}

// DefaultNew new an instance with default value
func DefaultNew(appID, userID, hmacWay string) *SignatureParams {
	return &SignatureParams{
		SgnType:   "apisign",
		Version:   "2020",
		AppID:     appID,
		UserID:    userID,
		Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
		Nonce:     fmt.Sprintf("%d", rand.Int63n(math.MaxInt64)),
		SgnMethod: hmacWay,
	}
}

// toBeSignedString 获取待签名串
func (s *SignatureParams) toBeSignedString() []byte {
	buf := make([]byte, 0, 256)
	buf = append(buf, s.Version...)
	buf = append(buf, '.')
	buf = append(buf, s.AppID...)
	buf = append(buf, '.')
	buf = append(buf, s.UserID...)
	buf = append(buf, '.')
	buf = append(buf, s.Timestamp...)
	buf = append(buf, '.')
	buf = append(buf, s.Nonce...)
	buf = append(buf, '.')
	buf = append(buf, s.SgnMethod...)
	buf = append(buf, '.')
	buf = append(buf, s.SgnBody...)

	return buf
}

// SignedString 获取签名串
func (s *SignatureParams) SignedString(key, body []byte) error {
	if body != nil && len(body) > 0 {
		sgnBody, err := SignedString(s.SgnMethod, key, body)
		if err != nil {
			return err
		}
		s.SgnBody = string(sgnBody)
	}
	tobss := s.toBeSignedString()
	sgn, err := SignedString(s.SgnMethod, key, tobss)
	if err != nil {
		return err
	}
	s.Signature = sgn
	return nil
}

// SignedString 获得签名串
func SignedString(hmacWay string, key, p []byte) (string, error) {
	var h hash.Hash
	if hmacWay == "sha256" {
		h = hmac.New(sha256.New, key)
	} else {
		h = hmac.New(sha1.New, key)
	}
	_, err := h.Write(p)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
