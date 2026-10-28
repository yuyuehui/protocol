package assistant

import (
	"errors"
	"testing"
)

func validFrame() *AssistantStreamNotification {
	return &AssistantStreamNotification{TaskID: "task", RunID: "run", StreamEpoch: 1, BaseRevision: 1, StreamSeq: 1, BlockID: "body", Utf8Offset: uint64(len("中文回答")), TextDelta: "你好"}
}

func TestApplyTextDelta(t *testing.T) {
	s := validSnapshot()
	frame := validFrame()
	next, applied, err := ApplyTextDelta(s, frame)
	if err != nil || !applied {
		t.Fatalf("增量失败: %v", err)
	}
	if next.Blocks[0].GetText().Text != "中文回答你好" || s.Blocks[0].GetText().Text != "中文回答" || s.LastStreamSeq != 0 {
		t.Fatal("增量内容错误或修改了输入")
	}
	duplicate, applied, err := ApplyTextDelta(next, frame)
	if err != nil || applied || duplicate.Blocks[0].GetText().Text != "中文回答你好" {
		t.Fatal("重复帧未去重")
	}
}

func TestRejectStreamFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*AssistantElem, *AssistantStreamNotification)
		want   error
	}{
		{"缺片", func(_ *AssistantElem, f *AssistantStreamNotification) { f.StreamSeq = 2 }, ErrStreamGap},
		{"旧代次", func(_ *AssistantElem, f *AssistantStreamNotification) { f.StreamEpoch = 2 }, ErrStreamMismatch},
		{"旧版本", func(_ *AssistantElem, f *AssistantStreamNotification) { f.BaseRevision = 2 }, ErrStreamMismatch},
		{"跨任务", func(_ *AssistantElem, f *AssistantStreamNotification) { f.TaskID = "other" }, ErrStreamMismatch},
		{"终态", func(s *AssistantElem, _ *AssistantStreamNotification) { s.Status = RunStatus_RUN_STATUS_COMPLETED }, ErrStreamClosed},
		{"撤回", func(s *AssistantElem, _ *AssistantStreamNotification) { s.Status = RunStatus_RUN_STATUS_REVOKED }, ErrStreamClosed},
		{"未知", func(s *AssistantElem, _ *AssistantStreamNotification) { s.Status = RunStatus_RUN_STATUS_UNKNOWN }, ErrStreamClosed},
		{"字符偏移", func(_ *AssistantElem, f *AssistantStreamNotification) { f.Utf8Offset = 4 }, ErrStreamOffset},
		{"无效UTF8", func(_ *AssistantElem, f *AssistantStreamNotification) { f.TextDelta = string([]byte{0xff}) }, ErrStreamOffset},
		{"未知块", func(_ *AssistantElem, f *AssistantStreamNotification) { f.BlockID = "missing" }, ErrStreamMismatch},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, f := validSnapshot(), validFrame()
			tt.mutate(s, f)
			_, applied, err := ApplyTextDelta(s, f)
			if applied || !errors.Is(err, tt.want) {
				t.Fatalf("期望 %v，实际 %v", tt.want, err)
			}
		})
	}
}
