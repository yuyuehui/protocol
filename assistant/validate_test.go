package assistant

import (
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func validSnapshot() *AssistantElem {
	return &AssistantElem{SchemaVersion: SchemaVersion, TaskID: "task", RunID: "run", SourceID: "source", Revision: 1, StreamEpoch: 1, Status: RunStatus_RUN_STATUS_RUNNING, Blocks: []*Block{{BlockID: "body", Content: &Block_Text{Text: &TextBlock{Text: "中文回答"}}}}}
}

func TestPublicSnapshotRoundTrip(t *testing.T) {
	s := validSnapshot()
	if err := ValidatePublicSnapshot(s); err != nil {
		t.Fatal(err)
	}
	data, err := protojson.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AssistantElem
	if err := protojson.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(s, &decoded) {
		t.Fatal("公开快照序列化不一致")
	}
}

func TestRejectInvalidSnapshots(t *testing.T) {
	cases := map[string]func(*AssistantElem){
		"版本未知":   func(s *AssistantElem) { s.SchemaVersion++ },
		"状态未知":   func(s *AssistantElem) { s.Status = RunStatus(99) },
		"缺少来源":   func(s *AssistantElem) { s.SourceID = "" },
		"重复块":    func(s *AssistantElem) { s.Blocks = append(s.Blocks, s.Blocks[0]) },
		"空块":     func(s *AssistantElem) { s.Blocks = append(s.Blocks, nil) },
		"空正文载荷":  func(s *AssistantElem) { s.Blocks[0].Content = &Block_Text{} },
		"无效UTF8": func(s *AssistantElem) { s.Blocks[0].GetText().Text = string([]byte{0xff}) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := validSnapshot()
			mutate(s)
			if ValidatePublicSnapshot(s) == nil {
				t.Fatal("无效快照未被拒绝")
			}
		})
	}
	if ValidatePublicSnapshot(nil) == nil {
		t.Fatal("nil快照未被拒绝")
	}
}

func TestTerminalStatus(t *testing.T) {
	for _, status := range []RunStatus{RunStatus_RUN_STATUS_COMPLETED, RunStatus_RUN_STATUS_FAILED, RunStatus_RUN_STATUS_STOPPED, RunStatus_RUN_STATUS_REVOKED} {
		if !IsTerminal(status) {
			t.Fatalf("终态未识别: %v", status)
		}
	}
	if !IsTerminal(RunStatus_RUN_STATUS_NOT_ACCEPTED) {
		t.Fatal("未受理应为终态")
	}
	if IsTerminal(RunStatus_RUN_STATUS_UNKNOWN) || IsTerminal(RunStatus_RUN_STATUS_STOPPING) {
		t.Fatal("未知或停止中不能当作已结束")
	}
}

func noticeOnly() *AssistantElem {
	return &AssistantElem{SchemaVersion: SchemaVersion, SourceID: "msg_1", Revision: 1, Status: RunStatus_RUN_STATUS_NOT_ACCEPTED, Blocks: []*Block{{BlockID: "notice", Content: &Block_Notice{Notice: &NoticeBlock{Code: "bot_offline", Summary: "助手当前不在线"}}}}}
}

func TestNoticeOnlySnapshot(t *testing.T) {
	if err := ValidatePublicSnapshot(noticeOnly()); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*AssistantElem){
		"缺少来源":   func(s *AssistantElem) { s.SourceID = "" },
		"非提示块":   func(s *AssistantElem) { s.Blocks = append(s.Blocks, validSnapshot().Blocks[0]) },
		"无块":     func(s *AssistantElem) { s.Blocks = nil },
		"有运行无任务": func(s *AssistantElem) { s.RunID = "run" },
		"有流序号":   func(s *AssistantElem) { s.LastStreamSeq = 1 },
		"非终态无任务": func(s *AssistantElem) { s.Status = RunStatus_RUN_STATUS_FAILED },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := noticeOnly()
			mutate(s)
			if ValidatePublicSnapshot(s) == nil {
				t.Fatal("无效提示消息未被拒绝")
			}
		})
	}
}

func interactionSnapshot(ref *InteractionRef) *AssistantElem {
	s := validSnapshot()
	s.Status = RunStatus_RUN_STATUS_WAITING
	s.Blocks = append(s.Blocks, &Block{BlockID: "card", Content: &Block_Interaction{Interaction: ref}})
	return s
}

func TestInteractionState(t *testing.T) {
	base := func() *InteractionRef {
		return &InteractionRef{InteractionID: "i1", Kind: InteractionKind_INTERACTION_KIND_EXECUTION_APPROVAL, Revision: 1, ExpiresAt: 10, Summary: "等待执行审批", State: InteractionState_INTERACTION_STATE_PENDING}
	}
	valid := map[string]func(*InteractionRef){
		"待处理": func(*InteractionRef) {},
		"已允许": func(r *InteractionRef) {
			r.State, r.Outcome, r.ActorUserID, r.DecidedAt = InteractionState_INTERACTION_STATE_ACCEPTED, InteractionOutcome_INTERACTION_OUTCOME_APPROVED, "u1", 5
		},
		"用户取消": func(r *InteractionRef) {
			r.State, r.Outcome, r.ActorUserID, r.DecidedAt = InteractionState_INTERACTION_STATE_SUBMITTED, InteractionOutcome_INTERACTION_OUTCOME_CANCELLED, "u1", 5
		},
		"已过期": func(r *InteractionRef) { r.State, r.DecidedAt = InteractionState_INTERACTION_STATE_EXPIRED, 10 },
		"停止作废": func(r *InteractionRef) {
			r.State, r.ActorUserID, r.DecidedAt = InteractionState_INTERACTION_STATE_CANCELLED, "u1", 6
		},
	}
	for name, mutate := range valid {
		ref := base()
		mutate(ref)
		if err := ValidatePublicSnapshot(interactionSnapshot(ref)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	invalid := map[string]func(*InteractionRef){
		"未设状态":   func(r *InteractionRef) { r.State = InteractionState_INTERACTION_STATE_UNSPECIFIED },
		"待处理带结果": func(r *InteractionRef) { r.Outcome = InteractionOutcome_INTERACTION_OUTCOME_APPROVED },
		"提交缺处理人": func(r *InteractionRef) {
			r.State, r.Outcome, r.DecidedAt = InteractionState_INTERACTION_STATE_SUBMITTED, InteractionOutcome_INTERACTION_OUTCOME_APPROVED, 5
		},
		"结果与类别不符": func(r *InteractionRef) {
			r.State, r.Outcome, r.ActorUserID, r.DecidedAt = InteractionState_INTERACTION_STATE_SUBMITTED, InteractionOutcome_INTERACTION_OUTCOME_ANSWERED, "u1", 5
		},
		"过期带处理人": func(r *InteractionRef) {
			r.State, r.ActorUserID, r.DecidedAt = InteractionState_INTERACTION_STATE_EXPIRED, "u1", 10
		},
	}
	for name, mutate := range invalid {
		ref := base()
		mutate(ref)
		if ValidatePublicSnapshot(interactionSnapshot(ref)) == nil {
			t.Fatalf("%s: 未被拒绝", name)
		}
	}
}

func TestPublicTextSafety(t *testing.T) {
	for _, text := range []string{"读取附件", "销售表.xlsx · Sheet1", "查询日程 · 3 条", "Read file report.pdf"} {
		if UnsafePublicText(text) {
			t.Fatalf("误判: %s", text)
		}
	}
	for _, text := range []string{"/Users/alice/report.pdf", `C:\work\a.txt`, "~/notes.md", "https://example.com/a", "token=abc", "sk-abcdefghijklmnopqrstuv", "rm -rf build", `\\server\share\a`} {
		if !UnsafePublicText(text) {
			t.Fatalf("漏判: %s", text)
		}
	}
	s := progressSnapshot()
	s.Blocks[1].GetProgress().Steps = []*ProcessStep{{StepID: "s1", Kind: StepKind_STEP_KIND_TOOL, Status: StepStatus_STEP_STATUS_RUNNING, Title: "读取文件", Detail: "/home/a/secret.txt", StartedAt: 1}}
	if CheckPublicProgress(s) == nil {
		t.Fatal("步骤中的本机路径未被拒绝")
	}
	if InboundAttachmentID("m1", 0) == InboundAttachmentID("m1", 1) || InboundAttachmentID("m1", 0) != InboundAttachmentID("m1", 0) {
		t.Fatal("附件 ID 不稳定或冲突")
	}
}
