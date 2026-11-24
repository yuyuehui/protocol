package assistant

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const SchemaVersion uint32 = 1

// 公开过程的结构上限；超出时由发送方归组，不能截断后冒充完整过程。
const (
	MaxProcessSteps     = 50
	MaxStepIDBytes      = 64
	MaxStepTitleRunes   = 64
	MaxStepDetailRunes  = 200
	MaxStepGroupBytes   = 64
	MaxErrorCodeBytes   = 64
	MaxPhaseBytes       = 64
	MaxSummaryRunes     = 200
	MaxNoticeCodeBytes  = 64
	MaxNoticeSummaryLen = 500
	// 云文档卡片：一条消息最多几张、标题最多几个字符（与 IM 文档卡片消息一致）
	MaxDocCards          = 10
	MaxDocCardTitleRunes = 200
	// 回复提醒的人：最多几个、每个用户 ID 最长几个字节、显示名最多几个字符
	MaxMentions         = 10
	MaxMentionIDBytes   = 64
	MaxMentionNameRunes = 64
	// 交互块里问答的回答摘要最多几个字符（InteractionRef.answerSummary）
	MaxAnswerSummaryRunes = 200
)

var docCardTID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidDocCard 云文档卡片的字段形状，规则与 IM 文档卡片消息（自定义消息 110）一致
func ValidDocCard(c *DocCardBlock) bool {
	if c == nil || !docCardTID.MatchString(c.Tid) || strings.TrimSpace(c.Title) == "" || !utf8.ValidString(c.Title) || utf8.RuneCountInString(c.Title) > MaxDocCardTitleRunes {
		return false
	}
	switch c.DocType {
	case "doc", "excel", "board":
	default:
		return false
	}
	for _, r := range c.Title {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// 提议卡字段上限
const (
	MaxProposalIDBytes       = 64
	MaxProposalNameRunes     = 64
	MaxProposalScheduleRunes = 64
	MaxProposalSummaryRunes  = 200
)

// ValidScheduleProposal 定时任务提议卡的字段形状；提议本身存不存在、谁能创建由 im-server 在创建时核对
func ValidScheduleProposal(p *ScheduleProposalBlock) bool {
	if p == nil || p.ProposalID == "" || len(p.ProposalID) > MaxProposalIDBytes || p.InitiatorUserID == "" || p.ExpiresAt <= 0 {
		return false
	}
	for _, f := range []struct {
		text string
		max  int
		need bool
	}{{p.Name, MaxProposalNameRunes, true}, {p.ScheduleText, MaxProposalScheduleRunes, true}, {p.PromptSummary, MaxProposalSummaryRunes, false}} {
		if !utf8.ValidString(f.text) || utf8.RuneCountInString(f.text) > f.max || (f.need && strings.TrimSpace(f.text) == "") {
			return false
		}
	}
	switch p.DeliveryKind {
	case "single", "group":
		return p.DeliveryTargetID != ""
	default:
		return false
	}
}

// MentionDisplayName 把发送人昵称整理成提醒的显示名：去掉无效编码和控制字符、首尾空白，最多 MaxMentionNameRunes 个字符
func MentionDisplayName(name string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.ToValidUTF8(name, ""))
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > MaxMentionNameRunes {
		name = strings.TrimSpace(string([]rune(name)[:MaxMentionNameRunes]))
	}
	return name
}

// validMentions 设了就至少一个，不重复，不超过上限；显示名要么没有，要么和用户 ID 一样多且都整理过
func validMentions(m *Mentions) bool {
	if len(m.UserIDs) == 0 || len(m.UserIDs) > MaxMentions {
		return false
	}
	if len(m.DisplayNames) != 0 && len(m.DisplayNames) != len(m.UserIDs) {
		return false
	}
	for _, name := range m.DisplayNames {
		if MentionDisplayName(name) != name {
			return false
		}
	}
	seen := make(map[string]struct{}, len(m.UserIDs))
	for _, id := range m.UserIDs {
		if _, exists := seen[id]; exists || id == "" || len(id) > MaxMentionIDBytes {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

// 附件类别（AttachmentBlock.kind），服务端按真实字节判断
const (
	AttachmentKindImage = "image"
	AttachmentKindVideo = "video"
	AttachmentKindAudio = "audio"
	AttachmentKindFile  = "file"
	// IM 文件对象名最长几个字节（与 IM 上传的对象名一致）
	MaxIMObjectBytes = 1024
)

// 客户端直接拼 <API 地址>/object/<imObject>，对象名只用不需要转义的字符
var imObjectSegment = regexp.MustCompile(`^[A-Za-z0-9._@-]+$`)

// validAttachmentFile 服务端写的 IM 文件字段：类别和对象名要么都有要么都没有；宽高只有图片带，两个都带且大于 0
func validAttachmentFile(a *AttachmentBlock) bool {
	if a.Kind == nil && a.ImObject == nil {
		return a.Width == nil && a.Height == nil
	}
	if a.Kind == nil || a.ImObject == nil || !ValidIMObject(*a.ImObject) {
		return false
	}
	switch *a.Kind {
	case AttachmentKindImage:
		return a.Width != nil && a.Height != nil && *a.Width > 0 && *a.Height > 0
	case AttachmentKindVideo, AttachmentKindAudio, AttachmentKindFile:
		return a.Width == nil && a.Height == nil
	}
	return false
}

// ValidIMObject IM 文件对象名：用 / 分段，每段只有字母、数字和 . _ @ -，不能是 . 或 ..
func ValidIMObject(name string) bool {
	if len(name) > MaxIMObjectBytes {
		return false
	}
	for _, seg := range strings.Split(name, "/") {
		if seg == "." || seg == ".." || !imObjectSegment.MatchString(seg) {
			return false
		}
	}
	return true
}

var ErrInvalidSnapshot = errors.New("invalid assistant snapshot")

// ValidatePublicSnapshot 校验公开消息的结构，不替代来源授权及安全投影。
func ValidatePublicSnapshot(s *AssistantElem) error {
	if s == nil || s.SchemaVersion != SchemaVersion || s.SourceID == "" || s.Revision == 0 {
		return ErrInvalidSnapshot
	}
	if s.Status < RunStatus_RUN_STATUS_ACCEPTED || s.Status > RunStatus_RUN_STATUS_NOT_ACCEPTED {
		return ErrInvalidSnapshot
	}
	if len(s.Phase) > MaxPhaseBytes || utf8.RuneCountInString(s.Summary) > MaxSummaryRunes || !utf8.ValidString(s.Summary) {
		return ErrInvalidSnapshot
	}
	if s.Status == RunStatus_RUN_STATUS_NOT_ACCEPTED {
		// 仅提示消息：没有运行可以流式、交互或重放，只允许 notice 块。
		if len(s.Blocks) == 0 || s.LastStreamSeq != 0 {
			return ErrInvalidSnapshot
		}
		for _, b := range s.Blocks {
			if b.GetNotice() == nil {
				return fmt.Errorf("%w: not accepted message allows notice blocks only", ErrInvalidSnapshot)
			}
		}
		if s.TaskID == "" && (s.RunID != "" || s.StreamEpoch != 0) {
			return ErrInvalidSnapshot
		}
	} else if s.TaskID == "" || s.RunID == "" || s.StreamEpoch == 0 {
		return ErrInvalidSnapshot
	}
	if s.Mentions != nil && !validMentions(s.Mentions) {
		return fmt.Errorf("%w: invalid mentions", ErrInvalidSnapshot)
	}
	seen := make(map[string]struct{}, len(s.Blocks))
	docCards := 0
	proposals := 0
	for _, b := range s.Blocks {
		if b == nil || b.BlockID == "" || b.Content == nil {
			return ErrInvalidSnapshot
		}
		if _, exists := seen[b.BlockID]; exists {
			return fmt.Errorf("%w: duplicate blockID", ErrInvalidSnapshot)
		}
		seen[b.BlockID] = struct{}{}
		switch c := b.Content.(type) {
		case *Block_Text:
			if c == nil || c.Text == nil || !utf8.ValidString(c.Text.Text) {
				return ErrInvalidSnapshot
			}
		case *Block_Progress:
			if c == nil || c.Progress == nil {
				return ErrInvalidSnapshot
			}
			if err := validateProgress(c.Progress); err != nil {
				return err
			}
		case *Block_Attachment:
			if c == nil || c.Attachment == nil || c.Attachment.ResourceID == "" || c.Attachment.ContentVersion == "" || c.Attachment.Size < 0 || !validAttachmentFile(c.Attachment) {
				return ErrInvalidSnapshot
			}
		case *Block_Sources:
			if c == nil || c.Sources == nil {
				return ErrInvalidSnapshot
			}
			for _, source := range c.Sources.Sources {
				if source == nil || source.SourceID == "" || source.ConversationID == "" || source.ContentVersion == "" {
					return ErrInvalidSnapshot
				}
			}
		case *Block_Interaction:
			if c == nil || c.Interaction == nil || c.Interaction.InteractionID == "" || c.Interaction.Revision == 0 || c.Interaction.ExpiresAt <= 0 || c.Interaction.Kind < InteractionKind_INTERACTION_KIND_QUESTION || c.Interaction.Kind > InteractionKind_INTERACTION_KIND_DRAFT_CONFIRMATION {
				return ErrInvalidSnapshot
			}
			if err := validateInteractionState(c.Interaction); err != nil {
				return err
			}
		case *Block_Receipt:
			if c == nil || c.Receipt == nil || c.Receipt.DeliveryID == "" || c.Receipt.Status < DeliveryStatus_DELIVERY_STATUS_ACCEPTED || c.Receipt.Status > DeliveryStatus_DELIVERY_STATUS_UNKNOWN {
				return ErrInvalidSnapshot
			}
		case *Block_Notice:
			if c == nil || c.Notice == nil || c.Notice.Code == "" || len(c.Notice.Code) > MaxNoticeCodeBytes || !utf8.ValidString(c.Notice.Summary) || utf8.RuneCountInString(c.Notice.Summary) > MaxNoticeSummaryLen {
				return ErrInvalidSnapshot
			}
		case *Block_DocCard:
			if docCards++; c == nil || !ValidDocCard(c.DocCard) || docCards > MaxDocCards {
				return ErrInvalidSnapshot
			}
		case *Block_ScheduleProposal:
			if proposals++; c == nil || !ValidScheduleProposal(c.ScheduleProposal) || proposals > 1 {
				return ErrInvalidSnapshot
			}
		default:
			return ErrInvalidSnapshot
		}
	}
	return nil
}

func validateProgress(p *ProgressBlock) error {
	if len(p.Phase) > MaxPhaseBytes || !utf8.ValidString(p.Summary) || utf8.RuneCountInString(p.Summary) > MaxSummaryRunes {
		return ErrInvalidSnapshot
	}
	if len(p.Steps) > MaxProcessSteps {
		return fmt.Errorf("%w: too many process steps", ErrInvalidSnapshot)
	}
	seen := make(map[string]struct{}, len(p.Steps))
	var last int64
	for _, step := range p.Steps {
		if step == nil || step.StepID == "" || len(step.StepID) > MaxStepIDBytes {
			return ErrInvalidSnapshot
		}
		if _, exists := seen[step.StepID]; exists {
			return fmt.Errorf("%w: duplicate stepID", ErrInvalidSnapshot)
		}
		seen[step.StepID] = struct{}{}
		if step.Kind < StepKind_STEP_KIND_THINKING || step.Kind > StepKind_STEP_KIND_COMPACTION || step.Status < StepStatus_STEP_STATUS_RUNNING || step.Status > StepStatus_STEP_STATUS_CANCELLED {
			return ErrInvalidSnapshot
		}
		if step.Title == "" || !utf8.ValidString(step.Title) || utf8.RuneCountInString(step.Title) > MaxStepTitleRunes || !utf8.ValidString(step.Detail) || utf8.RuneCountInString(step.Detail) > MaxStepDetailRunes || len(step.GroupKey) > MaxStepGroupBytes || len(step.ErrorCode) > MaxErrorCodeBytes {
			return ErrInvalidSnapshot
		}
		if step.StartedAt <= 0 || step.StartedAt < last {
			return fmt.Errorf("%w: steps must be ordered by startedAt", ErrInvalidSnapshot)
		}
		last = step.StartedAt
		if step.Status == StepStatus_STEP_STATUS_RUNNING {
			if step.EndedAt != 0 {
				return ErrInvalidSnapshot
			}
		} else if step.EndedAt < step.StartedAt {
			return ErrInvalidSnapshot
		}
		if step.ErrorCode != "" && step.Status != StepStatus_STEP_STATUS_FAILED && step.Status != StepStatus_STEP_STATUS_CANCELLED {
			return ErrInvalidSnapshot
		}
	}
	return nil
}

// validateInteractionState 只校验状态字段的自洽，记录一致性由服务端比对交互存储。
func validateInteractionState(ref *InteractionRef) error {
	decided := ref.ActorUserID != "" || ref.Outcome != InteractionOutcome_INTERACTION_OUTCOME_UNSPECIFIED
	switch ref.State {
	case InteractionState_INTERACTION_STATE_PENDING:
		if decided || ref.DecidedAt != 0 {
			return ErrInvalidSnapshot
		}
	case InteractionState_INTERACTION_STATE_SUBMITTED, InteractionState_INTERACTION_STATE_ACCEPTED:
		if ref.ActorUserID == "" || ref.DecidedAt <= 0 || !OutcomeAllowed(ref.Kind, ref.Outcome) {
			return ErrInvalidSnapshot
		}
	case InteractionState_INTERACTION_STATE_EXPIRED:
		if decided || ref.DecidedAt <= 0 {
			return ErrInvalidSnapshot
		}
	case InteractionState_INTERACTION_STATE_CANCELLED:
		// 运行停止导致作废：处理人可为发起停止的用户，但没有用户决定。
		if ref.Outcome != InteractionOutcome_INTERACTION_OUTCOME_UNSPECIFIED || ref.DecidedAt <= 0 {
			return ErrInvalidSnapshot
		}
	default:
		return ErrInvalidSnapshot
	}
	return validateInteractionExtras(ref)
}

// validateInteractionExtras 服务端给出的题型、回答摘要和处理来源要和类别、状态对得上：题型只有问答有；
// 回答摘要只有已回答的问答有；已在执行端处理的是受理了的，已撤销的是作废了的、没有处理人。
func validateInteractionExtras(ref *InteractionRef) error {
	if ref.QuestionType != nil && (ref.Kind != InteractionKind_INTERACTION_KIND_QUESTION ||
		(ref.GetQuestionType() != InteractionQuestionType_INTERACTION_QUESTION_TYPE_CONFIRM && ref.GetQuestionType() != InteractionQuestionType_INTERACTION_QUESTION_TYPE_CHOICE)) {
		return ErrInvalidSnapshot
	}
	if ref.AnswerSummary != nil {
		summary := ref.GetAnswerSummary()
		answered := ref.State == InteractionState_INTERACTION_STATE_SUBMITTED || ref.State == InteractionState_INTERACTION_STATE_ACCEPTED
		if ref.Kind != InteractionKind_INTERACTION_KIND_QUESTION || !answered || ref.Outcome != InteractionOutcome_INTERACTION_OUTCOME_ANSWERED ||
			strings.TrimSpace(summary) == "" || !utf8.ValidString(summary) || utf8.RuneCountInString(summary) > MaxAnswerSummaryRunes {
			return ErrInvalidSnapshot
		}
	}
	if ref.ResolvedBy != nil {
		switch ref.GetResolvedBy() {
		case InteractionResolver_INTERACTION_RESOLVER_EXECUTOR:
			if ref.State != InteractionState_INTERACTION_STATE_ACCEPTED {
				return ErrInvalidSnapshot
			}
		case InteractionResolver_INTERACTION_RESOLVER_REVOKED:
			if ref.State != InteractionState_INTERACTION_STATE_CANCELLED || ref.ActorUserID != "" {
				return ErrInvalidSnapshot
			}
		default:
			return ErrInvalidSnapshot
		}
	}
	return nil
}

// OutcomeAllowed 判断用户决定与交互类别是否匹配；取消适用于全部类别。
func OutcomeAllowed(kind InteractionKind, outcome InteractionOutcome) bool {
	if outcome == InteractionOutcome_INTERACTION_OUTCOME_CANCELLED {
		return kind >= InteractionKind_INTERACTION_KIND_QUESTION && kind <= InteractionKind_INTERACTION_KIND_DRAFT_CONFIRMATION
	}
	switch kind {
	case InteractionKind_INTERACTION_KIND_QUESTION:
		return outcome == InteractionOutcome_INTERACTION_OUTCOME_ANSWERED
	case InteractionKind_INTERACTION_KIND_EXECUTION_APPROVAL:
		return outcome == InteractionOutcome_INTERACTION_OUTCOME_APPROVED || outcome == InteractionOutcome_INTERACTION_OUTCOME_REJECTED
	case InteractionKind_INTERACTION_KIND_DRAFT_CONFIRMATION:
		return outcome == InteractionOutcome_INTERACTION_OUTCOME_CONFIRMED
	default:
		return false
	}
}

// IsTerminal 未知执行状态保留核验通道，不视为执行完成。
func IsTerminal(status RunStatus) bool {
	switch status {
	case RunStatus_RUN_STATUS_COMPLETED, RunStatus_RUN_STATUS_FAILED, RunStatus_RUN_STATUS_STOPPED, RunStatus_RUN_STATUS_REVOKED, RunStatus_RUN_STATUS_NOT_ACCEPTED:
		return true
	default:
		return false
	}
}
