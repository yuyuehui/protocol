package assistant

import (
	"strings"
	"testing"
	"time"
)

func TestGatewayAssertionBinding(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	now := time.Now()
	body := []byte(`{"taskID":"task"}`)
	claims := GatewayClaims{ApplicationID: "app", BotID: "bot", ConnectionID: "connection", ActionID: "action", Method: "POST", Path: "/assistant/internal/snapshot", BodyHash: GatewayBodyHash(body), IssuedAt: now.Unix(), Nonce: "0123456789abcdef"}
	signed, err := SignGatewayAssertion(key, claims)
	if err != nil {
		t.Fatal(err)
	}
	got, err := VerifyGatewayAssertion(key, signed, claims.Method, claims.Path, body, now)
	if err != nil || got != claims {
		t.Fatal("合法签名失败", err)
	}
	for _, tc := range []struct {
		name, method, path string
		body               []byte
		now                time.Time
	}{
		{"正文篡改", "POST", claims.Path, []byte(`{}`), now},
		{"路由替换", "POST", "/assistant/internal/finish", body, now},
		{"方法替换", "GET", claims.Path, body, now},
		{"过期", "POST", claims.Path, body, now.Add(61 * time.Second)},
		{"未来签发", "POST", claims.Path, body, now.Add(-10 * time.Second)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := VerifyGatewayAssertion(key, signed, tc.method, tc.path, tc.body, tc.now); err == nil {
				t.Fatal("篡改请求未拒绝")
			}
		})
	}
	if _, err := VerifyGatewayAssertion([]byte(strings.Repeat("x", 32)), signed, claims.Method, claims.Path, body, now); err == nil {
		t.Fatal("错误密钥未拒绝")
	}
}
