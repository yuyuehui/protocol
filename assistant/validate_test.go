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
	if IsTerminal(RunStatus_RUN_STATUS_UNKNOWN) || IsTerminal(RunStatus_RUN_STATUS_STOPPING) {
		t.Fatal("未知或停止中不能当作已结束")
	}
}
