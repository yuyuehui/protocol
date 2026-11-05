package assistant

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
)

func validFrame() *AssistantStreamNotification {
	return &AssistantStreamNotification{TaskID: "task", RunID: "run", StreamEpoch: 1, BaseRevision: 1, StreamSeq: 1, BlockID: "body", Utf8Offset: uint64(len("中文回答")), Delta: &AssistantStreamNotification_TextDelta{TextDelta: "你好"}}
}

func TestApplyStreamFrame(t *testing.T) {
	s := validSnapshot()
	frame := validFrame()
	next, applied, err := ApplyStreamFrame(s, frame)
	if err != nil || !applied {
		t.Fatalf("增量失败: %v", err)
	}
	if next.Blocks[0].GetText().Text != "中文回答你好" || s.Blocks[0].GetText().Text != "中文回答" || s.LastStreamSeq != 0 {
		t.Fatal("增量内容错误或修改了输入")
	}
	duplicate, applied, err := ApplyStreamFrame(next, frame)
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
		{"无效UTF8", func(_ *AssistantElem, f *AssistantStreamNotification) {
			f.Delta = &AssistantStreamNotification_TextDelta{TextDelta: string([]byte{0xff})}
		}, ErrStreamOffset},
		{"未知块", func(_ *AssistantElem, f *AssistantStreamNotification) { f.BlockID = "missing" }, ErrStreamMismatch},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, f := validSnapshot(), validFrame()
			tt.mutate(s, f)
			_, applied, err := ApplyStreamFrame(s, f)
			if applied || !errors.Is(err, tt.want) {
				t.Fatalf("期望 %v，实际 %v", tt.want, err)
			}
		})
	}
}

func progressSnapshot() *AssistantElem {
	s := validSnapshot()
	s.Blocks = append(s.Blocks, &Block{BlockID: "process", Content: &Block_Progress{Progress: &ProgressBlock{Phase: "thinking"}}})
	return s
}

func progressFrame(seq uint64, step *ProcessStep) *AssistantStreamNotification {
	return &AssistantStreamNotification{TaskID: "task", RunID: "run", StreamEpoch: 1, BaseRevision: 1, StreamSeq: seq, BlockID: "process", Delta: &AssistantStreamNotification_Progress{Progress: &ProgressDelta{Summary: "正在读取附件", Step: step}}}
}

func TestApplyProgressFrames(t *testing.T) {
	s := progressSnapshot()
	read := &ProcessStep{StepID: "s2", Kind: StepKind_STEP_KIND_READ_ATTACHMENT, Status: StepStatus_STEP_STATUS_RUNNING, Title: "读取附件", StartedAt: 2000}
	next, applied, err := ApplyStreamFrame(s, progressFrame(1, read))
	if err != nil || !applied {
		t.Fatalf("进度帧失败: %v", err)
	}
	think := &ProcessStep{StepID: "s1", Kind: StepKind_STEP_KIND_THINKING, Status: StepStatus_STEP_STATUS_COMPLETED, Title: "思考", StartedAt: 1000, EndedAt: 1500}
	next, _, err = ApplyStreamFrame(next, progressFrame(2, think))
	if err != nil {
		t.Fatal(err)
	}
	done := &ProcessStep{StepID: "s2", Kind: StepKind_STEP_KIND_READ_ATTACHMENT, Status: StepStatus_STEP_STATUS_COMPLETED, Title: "读取附件", Detail: "销售表.xlsx", StartedAt: 2000, EndedAt: 2600}
	next, _, err = ApplyStreamFrame(next, progressFrame(3, done))
	if err != nil {
		t.Fatal(err)
	}
	progress := next.Blocks[1].GetProgress()
	if len(progress.Steps) != 2 || progress.Steps[0].StepID != "s1" || progress.Steps[1].Status != StepStatus_STEP_STATUS_COMPLETED || progress.Summary != "正在读取附件" || progress.Phase != "thinking" || next.LastStreamSeq != 3 {
		t.Fatalf("步骤合并错误: %+v", progress)
	}
	text := validFrame()
	text.StreamSeq = 4
	if _, applied, err := ApplyStreamFrame(next, text); err != nil || !applied {
		t.Fatalf("文本与进度共享序号失败: %v", err)
	}
}

func TestRejectProgressFrames(t *testing.T) {
	running := &ProcessStep{StepID: "s1", Kind: StepKind_STEP_KIND_TOOL, Status: StepStatus_STEP_STATUS_RUNNING, Title: "执行工具", StartedAt: 1000}
	cases := map[string]func(*AssistantStreamNotification){
		"文本块": func(f *AssistantStreamNotification) { f.BlockID = "body" },
		"带偏移": func(f *AssistantStreamNotification) { f.Utf8Offset = 1 },
		"空增量": func(f *AssistantStreamNotification) {
			f.Delta = &AssistantStreamNotification_Progress{Progress: &ProgressDelta{}}
		},
		"无载荷":    func(f *AssistantStreamNotification) { f.Delta = nil },
		"运行中有结束": func(f *AssistantStreamNotification) { f.GetProgress().Step.EndedAt = 2000 },
		"标题过长": func(f *AssistantStreamNotification) {
			f.GetProgress().Step.Title = strings.Repeat("长", MaxStepTitleRunes+1)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			step := proto.Clone(running).(*ProcessStep)
			f := progressFrame(1, step)
			mutate(f)
			if _, applied, err := ApplyStreamFrame(progressSnapshot(), f); err == nil || applied {
				t.Fatal("无效进度帧未被拒绝")
			}
		})
	}
	s := progressSnapshot()
	for i := 0; i < MaxProcessSteps; i++ {
		s.Blocks[1].GetProgress().Steps = append(s.Blocks[1].GetProgress().Steps, &ProcessStep{StepID: fmt.Sprintf("step-%d", i), Kind: StepKind_STEP_KIND_TOOL, Status: StepStatus_STEP_STATUS_COMPLETED, Title: "执行工具", StartedAt: int64(i + 1), EndedAt: int64(i + 1)})
	}
	if _, _, err := ApplyStreamFrame(s, progressFrame(1, &ProcessStep{StepID: "overflow", Kind: StepKind_STEP_KIND_TOOL, Status: StepStatus_STEP_STATUS_RUNNING, Title: "执行工具", StartedAt: 9999})); !errors.Is(err, ErrStreamFrame) {
		t.Fatalf("步骤超限未被拒绝: %v", err)
	}
}
