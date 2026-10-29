package assistant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
)

var ErrUnsafePublicText = errors.New("assistant public text contains private details")

// 公开过程会进入群消息；以下内容只能留在执行端，渠道插件须用同一组规则先行替换。
var unsafePublicPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://`),                                          // URL
	regexp.MustCompile(`(?i)\bwww\.[a-z0-9-]+\.`),                                             // 无协议 URL
	regexp.MustCompile(`(?i)(^|[\s"'(=:])(/users/|/home/|/root/|/tmp/|/var/|/etc/|/opt/|~/)`), // 类 Unix 本机路径
	regexp.MustCompile(`(?i)(^|[^a-z])[a-z]:[\\/]`),                                           // Windows 盘符路径
	regexp.MustCompile(`\\\\[^\\\s]+\\`),                                                      // UNC 路径
	regexp.MustCompile(`(?i)\b(api[_-]?key|secret|token|password|passwd|authorization|bearer)\b\s*[:=]`),
	regexp.MustCompile(`\b(sk-[A-Za-z0-9_-]{16,}|AKIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{20,}|xox[abprs]-[A-Za-z0-9-]{10,})`),
	regexp.MustCompile(`-----BEGIN [A-Z ]+-----`),
	regexp.MustCompile(`[A-Za-z0-9+/_=-]{40,}`), // 长串密钥或编码数据
	regexp.MustCompile(`(?i)(^|\s)(sudo|rm\s+-rf|curl\s|wget\s|powershell|cmd(\.exe)?\s+/c|bash\s+-c|sh\s+-c)\b`),
}

// UnsafePublicText 判断文本是否含本机路径、URL、命令行或疑似密钥。
func UnsafePublicText(s string) bool {
	for _, pattern := range unsafePublicPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
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
