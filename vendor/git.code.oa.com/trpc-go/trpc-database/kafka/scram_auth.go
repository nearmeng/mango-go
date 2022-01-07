// Package kafka 安全验证类
package kafka

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"git.code.oa.com/trpc-go/trpc-go/errs"
	"github.com/Shopify/sarama"
	"github.com/xdg-go/scram"
)

// SHA256 hash 协议
var SHA256 scram.HashGeneratorFcn = func() hash.Hash { return sha256.New() }

// SHA512 hash 协议
var SHA512 scram.HashGeneratorFcn = func() hash.Hash { return sha512.New() }

// LSCRAMClient scram 认证客户端配置
type LSCRAMClient struct {
	*scram.Client                    // 客户端
	*scram.ClientConversation        // 客户端会话层
	scram.HashGeneratorFcn           // hash值生成函数
	User                      string // 账号
	Password                  string // 密码
	Mechanism                 string // 加密协议类型
}

// Begin SCRAM 认证开始接口
func (s *LSCRAMClient) Begin(userName, password, authzID string) (err error) {
	s.Client, err = s.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	s.ClientConversation = s.Client.NewConversation()
	return nil
}

// Step SCRAM 认证步骤接口
func (s *LSCRAMClient) Step(challenge string) (response string, err error) {
	response, err = s.ClientConversation.Step(challenge)
	return
}

// Done SCRAM 认证结束接口
func (s *LSCRAMClient) Done() bool {
	return s.ClientConversation.Done()
}

// config 配置 sarama 客户端
func (s *LSCRAMClient) config(config *sarama.Config) error {
	if s == nil {
		return nil
	}

	// s 不为nil，表示已经初始化过了
	if len(s.Mechanism) == 0 {
		return errs.NewFrameError(errs.RetClientRouteErr, "kafka scram_client.config failed, Mechanism.len=0")
	}

	config.Net.SASL.Enable = true
	config.Net.SASL.User = s.User
	config.Net.SASL.Password = s.Password
	config.Net.SASL.Mechanism = sarama.SASLMechanism(s.Mechanism)
	switch s.Mechanism {
	case sarama.SASLTypeSCRAMSHA512:
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &LSCRAMClient{HashGeneratorFcn: SHA512}
		}
	case sarama.SASLTypeSCRAMSHA256:
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &LSCRAMClient{HashGeneratorFcn: SHA256}
		}
	default:
		return errs.NewFrameError(errs.RetClientRouteErr, "kafka scram_client.config failed,x.mechanism "+
			"unknown("+s.Mechanism+")")
	}

	return nil
}

// Parse SCRAM 本地解析
func (s *LSCRAMClient) Parse(vals []string) {
	switch vals[0] {
	case "user":
		s.User = vals[1]
	case "password":
		s.Password = vals[1]
	case "mechanism":
		s.Mechanism = vals[1]
	default:
	}
	return
}
