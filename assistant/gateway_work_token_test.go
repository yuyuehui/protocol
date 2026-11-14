package assistant

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// 工作令牌摘要随断言签名：没带时与旧断言一致，带了必须是 64 位摘要，签名后改动验签失败。
func TestGatewayAssertionWorkTokenHash(t *testing.T) {
	signing := []byte(strings.Repeat("w", 32))
	now := time.Now()
	body := []byte(`{"taskID":"task"}`)
	claims := GatewayClaims{ApplicationID: "app", BotID: "bot", ConnectionID: "connection", ActionID: "action", Method: "POST", Path: "/assistant/internal/snapshot", BodyHash: GatewayBodyHash(body), IssuedAt: now.Unix(), Nonce: "0123456789abcdef"}
	signed, err := SignGatewayAssertion(signing, claims)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.Split(signed, ".")[0])
	if err != nil || strings.Contains(string(raw), "workTokenHash") {
		t.Fatal("没带工作令牌时断言应与旧版一致", err)
	}
	claims.WorkTokenHash = GatewayBodyHash([]byte("work-token"))
	if signed, err = SignGatewayAssertion(signing, claims); err != nil {
		t.Fatal(err)
	}
	got, err := VerifyGatewayAssertion(signing, signed, claims.Method, claims.Path, body, now)
	if err != nil || got != claims {
		t.Fatal("带工作令牌摘要的断言验签失败", err)
	}
	forged := claims
	forged.WorkTokenHash = GatewayBodyHash([]byte("other-token"))
	payload, err := json.Marshal(forged)
	if err != nil {
		t.Fatal(err)
	}
	tampered := base64.RawURLEncoding.EncodeToString(payload) + "." + strings.Split(signed, ".")[1]
	if _, err = VerifyGatewayAssertion(signing, tampered, claims.Method, claims.Path, body, now); err == nil {
		t.Fatal("改了工作令牌摘要的断言未拒绝")
	}
	for _, hash := range []string{"short", strings.Repeat("a", 65)} {
		claims.WorkTokenHash = hash
		if _, err = SignGatewayAssertion(signing, claims); err == nil {
			t.Fatal("摘要长度不对的断言不应签发", len(hash))
		}
	}
}
