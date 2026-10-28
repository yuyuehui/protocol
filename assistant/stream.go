package assistant

import (
	"errors"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"
)

var (
	ErrStreamMismatch = errors.New("assistant stream identity or revision mismatch")
	ErrStreamGap      = errors.New("assistant stream gap: snapshot required")
	ErrStreamClosed   = errors.New("assistant stream is not writable")
	ErrStreamOffset   = errors.New("assistant stream UTF-8 offset mismatch")
)

// ApplyTextDelta 返回独立的内存投影，不修改权威快照，不执行持久化或权限判定。
// 调用者必须先核验会话、消息以及连接代次；重复序号不重复应用。
func ApplyTextDelta(s *AssistantElem, frame *AssistantStreamNotification) (*AssistantElem, bool, error) {
	if err := ValidatePublicSnapshot(s); err != nil {
		return nil, false, err
	}
	if frame == nil || frame.TaskID != s.TaskID || frame.RunID != s.RunID || frame.StreamEpoch != s.StreamEpoch || frame.BaseRevision != s.Revision || frame.StreamSeq == 0 {
		return nil, false, ErrStreamMismatch
	}
	if s.Status != RunStatus_RUN_STATUS_RUNNING {
		return nil, false, ErrStreamClosed
	}
	if frame.StreamSeq <= s.LastStreamSeq {
		return proto.Clone(s).(*AssistantElem), false, nil
	}
	if s.LastStreamSeq == ^uint64(0) || frame.StreamSeq != s.LastStreamSeq+1 {
		return nil, false, ErrStreamGap
	}
	if frame.BlockID == "" || !utf8.ValidString(frame.TextDelta) {
		return nil, false, ErrStreamOffset
	}
	for i, block := range s.Blocks {
		if block.BlockID != frame.BlockID {
			continue
		}
		text := block.GetText()
		if text == nil || uint64(len(text.Text)) != frame.Utf8Offset {
			return nil, false, ErrStreamOffset
		}
		next := proto.Clone(s).(*AssistantElem)
		next.Blocks[i].GetText().Text += frame.TextDelta
		next.LastStreamSeq = frame.StreamSeq
		return next, true, nil
	}
	// 新块先随可靠快照建立，增量不能自行创造结构化内容。
	return nil, false, ErrStreamMismatch
}
