package assistant

import (
	"errors"
	"fmt"
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
)

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
	seen := make(map[string]struct{}, len(s.Blocks))
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
			if c == nil || c.Attachment == nil || c.Attachment.ResourceID == "" || c.Attachment.ContentVersion == "" || c.Attachment.Size < 0 {
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
