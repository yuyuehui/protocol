package assistant

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const GatewayAssertionHeader = "X-Assistant-Gateway-Assertion"

var ErrGatewayAssertion = errors.New("invalid assistant gateway assertion")

// GatewayClaims 仅由已认证的 Bot 网关构造，不能从插件载荷复制身份字段。
type GatewayClaims struct {
	ApplicationID string `json:"applicationID"`
	BotID         string `json:"botID"`
	ConnectionID  string `json:"connectionID"`
	ActionID      string `json:"actionID"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	BodyHash      string `json:"bodyHash"`
	IssuedAt      int64  `json:"issuedAt"`
	Nonce         string `json:"nonce"`
}

func GatewayBodyHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
func validGatewayClaims(c GatewayClaims) bool {
	return c.ApplicationID != "" && len(c.ApplicationID) <= 256 && c.BotID != "" && len(c.BotID) <= 256 && c.ConnectionID != "" && len(c.ConnectionID) <= 256 && c.ActionID != "" && len(c.ActionID) <= 256 && c.Method == "POST" && strings.HasPrefix(c.Path, "/assistant/internal/") && !strings.ContainsAny(c.Path, "?#") && len(c.BodyHash) == 64 && len(c.Nonce) >= 16 && len(c.Nonce) <= 128 && c.IssuedAt > 0
}
func SignGatewayAssertion(key []byte, c GatewayClaims) (string, error) {
	if len(key) < 32 || !validGatewayClaims(c) {
		return "", ErrGatewayAssertion
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// VerifyGatewayAssertion 只验证签名与请求绑定；调用者必须原子消费 nonce 并核验当前连接。
func VerifyGatewayAssertion(key []byte, assertion, method, path string, body []byte, now time.Time) (GatewayClaims, error) {
	var c GatewayClaims
	if len(key) < 32 || len(assertion) > 8192 {
		return c, ErrGatewayAssertion
	}
	parts := strings.Split(assertion, ".")
	if len(parts) != 2 {
		return c, ErrGatewayAssertion
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return c, ErrGatewayAssertion
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return c, ErrGatewayAssertion
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return c, ErrGatewayAssertion
	}
	if json.Unmarshal(raw, &c) != nil || !validGatewayClaims(c) {
		return GatewayClaims{}, ErrGatewayAssertion
	}
	if c.Method != method || c.Path != path || c.BodyHash != GatewayBodyHash(body) || c.IssuedAt < now.Unix()-60 || c.IssuedAt > now.Unix()+5 {
		return GatewayClaims{}, ErrGatewayAssertion
	}
	return c, nil
}
