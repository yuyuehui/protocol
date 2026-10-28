package assistant

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

const SchemaVersion uint32 = 1

var ErrInvalidSnapshot = errors.New("invalid assistant snapshot")

// ValidatePublicSnapshot 校验公开消息的结构，不替代来源授权及安全投影。
func ValidatePublicSnapshot(s *AssistantElem) error {
	if s == nil || s.SchemaVersion != SchemaVersion || s.TaskID == "" || s.RunID == "" || s.SourceID == "" || s.Revision == 0 || s.StreamEpoch == 0 {
		return ErrInvalidSnapshot
	}
	if s.Status < RunStatus_RUN_STATUS_ACCEPTED || s.Status > RunStatus_RUN_STATUS_REVOKED {
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
		case *Block_Receipt:
			if c == nil || c.Receipt == nil || c.Receipt.DeliveryID == "" || c.Receipt.Status < DeliveryStatus_DELIVERY_STATUS_ACCEPTED || c.Receipt.Status > DeliveryStatus_DELIVERY_STATUS_UNKNOWN {
				return ErrInvalidSnapshot
			}
		case *Block_Notice:
			if c == nil || c.Notice == nil || c.Notice.Code == "" {
				return ErrInvalidSnapshot
			}
		default:
			return ErrInvalidSnapshot
		}
	}
	return nil
}

// IsTerminal 未知执行状态保留核验通道，不视为执行完成。
func IsTerminal(status RunStatus) bool {
	switch status {
	case RunStatus_RUN_STATUS_COMPLETED, RunStatus_RUN_STATUS_FAILED, RunStatus_RUN_STATUS_STOPPED, RunStatus_RUN_STATUS_REVOKED:
		return true
	default:
		return false
	}
}
