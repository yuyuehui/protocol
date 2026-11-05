package assistant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var ErrUnsafePublicText = errors.New("assistant public text contains private details")

// 公开过程会进入群消息；以下内容只能留在执行端，渠道插件须用同一组规则先行替换。
// 中文没有词边界，前缀统一用“非 ASCII 字母数字”判断，避免“打开/Users/…”漏判。
var unsafePublicPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)[a-z][a-z0-9+.-]*://`), // URL
	regexp.MustCompile(`(?i)(^|[^a-z0-9.@_-])www\.[a-z0-9-]+\.`),
	// 不带协议的域名与 IP
	regexp.MustCompile(`(?i)(^|[^a-z0-9.@_-])([a-z0-9-]+\.)+(com|cn|net|org|io|dev|ai|app|co|me|info|top|xyz|cloud|tech|site|online|gov|edu|local|internal)(:\d+)?([/?#:]|[^a-z0-9.-]|$)`),
	regexp.MustCompile(`(^|[^0-9.])(\d{1,3}\.){3}\d{1,3}([^0-9.]|$)`),
	// 类 Unix 本机路径与相对路径
	regexp.MustCompile(`(?i)(^|[^a-z0-9_.~-])(/users/|/home/|/root/|/tmp/|/var/|/etc/|/opt/|/data/|/mnt/|/srv/|/usr/|/private/|/volumes/|/workspace/|~/|\.\.?/)`),
	regexp.MustCompile(`(?i)(^|[^a-z])[a-z]:[\\/]`), // Windows 盘符路径
	regexp.MustCompile(`\\\\[^\\\s]+\\`),            // UNC 路径
	regexp.MustCompile(`(?i)(^|[^a-z])(api[_ -]?key|secret|password|passwd|pwd|authorization|cookie)\s*[:=：]\s*\S`),
	regexp.MustCompile(`(密码|口令|密钥|私钥|秘钥|令牌)\s*[:=：]\s*\S`),
	regexp.MustCompile(`(?i)(^|[^a-z])bearer\s+[A-Za-z0-9._~+/-]{8,}`),
	regexp.MustCompile(`\b(sk-[A-Za-z0-9_-]{16,}|AKIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{20,}|xox[abprs]-[A-Za-z0-9-]{10,})`),
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.`), // JWT
	regexp.MustCompile(`-----BEGIN [A-Z ]+-----`),
	regexp.MustCompile(`(?i)(^|[^a-z0-9_])(sudo|rm\s+-rf|curl\s|wget\s|powershell|cmd(\.exe)?\s+/c|bash\s+-c|sh\s+-c)\b`),
}

// token 的值只有像密钥时才拦截，“Token: 10”这类用量统计不算。
var tokenAssignment = regexp.MustCompile(`(?i)(^|[^a-z])(access[_ -]?token|refresh[_ -]?token|token)\s*[:=：]\s*([A-Za-z0-9._~+/-]+)`)

// 长串候选：不含 _ - . 分隔，长文件名通常被分隔符切成短段。
var longOpaqueRun = regexp.MustCompile(`[A-Za-z0-9+/=]{32,}`)

// UnsafePublicText 判断文本是否含本机路径、URL、命令行或疑似密钥。
func UnsafePublicText(s string) bool {
	for _, pattern := range unsafePublicPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	for _, m := range tokenAssignment.FindAllStringSubmatch(s, -1) {
		if v := m[3]; len(v) >= 8 && strings.IndexFunc(v, unicode.IsLetter) >= 0 {
			return true
		}
	}
	for _, run := range longOpaqueRun.FindAllString(s, -1) {
		if opaqueRun(run) {
			return true
		}
	}
	return false
}

// opaqueRun 字母数字混排且数字占比不低于 1/8 视为密钥或编码数据；
// 含 + / 的 40 位以上串按 base64 处理。驼峰长文件名末尾带年份不会命中。
func opaqueRun(run string) bool {
	digits, letters := 0, 0
	for _, r := range run {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case unicode.IsLetter(r):
			letters++
		}
	}
	if digits == 0 || letters == 0 {
		return false
	}
	return digits*8 >= len(run) || (len(run) >= 40 && strings.ContainsAny(run, "+/"))
}

// CheckPublicProgress 检查快照中全部公开过程文本；结构校验仍由 ValidatePublicSnapshot 负责。
func CheckPublicProgress(s *AssistantElem) error {
	if s == nil {
		return nil
	}
	if UnsafePublicText(s.Summary) {
		return ErrUnsafePublicText
	}
	for _, b := range s.Blocks {
		if err := checkProgressText(b.GetProgress()); err != nil {
			return err
		}
	}
	return nil
}

// CheckPublicProgressDelta 检查单个进度增量，供流式入口在合并前拒绝。
func CheckPublicProgressDelta(d *ProgressDelta) error {
	if d == nil {
		return nil
	}
	p := &ProgressBlock{Summary: d.Summary}
	if d.Step != nil {
		p.Steps = []*ProcessStep{d.Step}
	}
	return checkProgressText(p)
}

func checkProgressText(p *ProgressBlock) error {
	if p == nil {
		return nil
	}
	if UnsafePublicText(p.Summary) {
		return ErrUnsafePublicText
	}
	for _, step := range p.Steps {
		if step != nil && (UnsafePublicText(step.Title) || UnsafePublicText(step.Detail) || UnsafePublicText(step.GroupKey)) {
			return ErrUnsafePublicText
		}
	}
	return nil
}

// InboundAttachmentID 由来源消息与附件序号派生稳定 ID：网关入站时下发，
// 服务端在创建任务时按同一规则登记输入资源，执行端不接触可直接下载的地址。
func InboundAttachmentID(serverMsgID string, index int) string {
	data, _ := json.Marshal([]any{"assistant-inbound-attachment", serverMsgID, index})
	sum := sha256.Sum256(data)
	return "att_" + hex.EncodeToString(sum[:16])
}
