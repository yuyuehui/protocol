package assistant

import (
	"errors"
	"sort"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"
)

var (
	ErrStreamMismatch = errors.New("assistant stream identity or revision mismatch")
	ErrStreamGap      = errors.New("assistant stream gap: snapshot required")
	ErrStreamClosed   = errors.New("assistant stream is not writable")
	ErrStreamOffset   = errors.New("assistant stream UTF-8 offset mismatch")
	ErrStreamFrame    = errors.New("assistant stream frame invalid")
)

// ApplyStreamFrame 返回独立的内存投影，不修改权威快照，不执行持久化或权限判定。
// 调用者必须先核验会话、消息以及连接代次；文本帧与进度帧共享序号，重复序号不重复应用。
func ApplyStreamFrame(s *AssistantElem, frame *AssistantStreamNotification) (*AssistantElem, bool, error) {
	if err := ValidatePublicSnapshot(s); err != nil {
		return nil, false, err
	}
	if frame == nil || s.TaskID == "" || frame.TaskID != s.TaskID || frame.RunID != s.RunID || frame.StreamEpoch != s.StreamEpoch || frame.BaseRevision != s.Revision || frame.StreamSeq == 0 {
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
	if frame.BlockID == "" {
		return nil, false, ErrStreamFrame
	}
	index := -1
	for i, block := range s.Blocks {
		if block.BlockID == frame.BlockID {
			index = i
			break
		}
	}
	// 新块先随可靠快照建立，增量不能自行创造结构化内容。
	if index < 0 {
		return nil, false, ErrStreamMismatch
	}
	next := proto.Clone(s).(*AssistantElem)
	switch delta := frame.Delta.(type) {
	case *AssistantStreamNotification_TextDelta:
		if !utf8.ValidString(delta.TextDelta) {
			return nil, false, ErrStreamOffset
		}
		text := next.Blocks[index].GetText()
		if text == nil || uint64(len(text.Text)) != frame.Utf8Offset {
			return nil, false, ErrStreamOffset
		}
		text.Text += delta.TextDelta
	case *AssistantStreamNotification_Progress:
		progress := next.Blocks[index].GetProgress()
		if progress == nil {
			return nil, false, ErrStreamMismatch
		}
		if frame.Utf8Offset != 0 || !applyProgressDelta(progress, delta.Progress) {
			return nil, false, ErrStreamFrame
		}
	default:
		return nil, false, ErrStreamFrame
	}
	next.LastStreamSeq = frame.StreamSeq
	// 合并后整体重验，步骤上限、长度与时间顺序的约束对增量同样生效。
	if ValidatePublicSnapshot(next) != nil {
		return nil, false, ErrStreamFrame
	}
	return next, true, nil
}

// applyProgressDelta 原地合并进度增量，空增量视为无效帧。
func applyProgressDelta(p *ProgressBlock, d *ProgressDelta) bool {
	if d == nil || (d.Phase == "" && d.Summary == "" && d.Step == nil && d.Counts == nil) {
		return false
	}
	if d.Phase != "" {
		p.Phase = d.Phase
	}
	if d.Summary != "" {
		p.Summary = d.Summary
	}
	if d.Counts != nil {
		p.Counts = proto.Clone(d.Counts).(*ProcessCounts)
	}
	if d.Step != nil {
		step := proto.Clone(d.Step).(*ProcessStep)
		replaced := false
		for i, existing := range p.Steps {
			if existing.GetStepID() == step.StepID {
				p.Steps[i] = step
				replaced = true
				break
			}
		}
		if !replaced {
			p.Steps = append(p.Steps, step)
		}
		sort.SliceStable(p.Steps, func(i, j int) bool { return p.Steps[i].StartedAt < p.Steps[j].StartedAt })
	}
	return true
}
