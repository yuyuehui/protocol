package assistant

import (
	"bytes"
	"fmt"
	"strings"
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

// 服务端按 EmitUnpopulated 序列化 215 / 2122，老客户端严格解析、认不出的字段会让整条显示不了：
// 后加进公开快照的字段必须有 presence（optional 或 oneof），不设时不输出。
func TestLaterSnapshotFieldsOmittedWhenUnset(t *testing.T) {
	snapshot := validSnapshot()
	snapshot.Blocks = append(snapshot.Blocks, &Block{BlockID: "ask", Content: &Block_Interaction{Interaction: &InteractionRef{InteractionID: "ask"}}})
	content, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"confirmerUserID", "docCard", "mentions"} {
		if bytes.Contains(content, []byte(field)) {
			t.Fatalf("没设的 %s 不应输出: %s", field, content)
		}
	}
	confirmer := "hirer"
	snapshot.Blocks[1].GetInteraction().ConfirmerUserID = &confirmer
	if content, _ = (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(snapshot); !bytes.Contains(content, []byte(`"confirmerUserID":"hirer"`)) {
		t.Fatalf("设了确认人应输出: %s", content)
	}
	snapshot.Mentions = &Mentions{UserIDs: []string{"user"}}
	if content, _ = (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(snapshot); !bytes.Contains(content, []byte(`"mentions":{"userIDs":["user"]}`)) {
		t.Fatalf("设了提醒的人应输出: %s", content)
	}
}

func TestMentions(t *testing.T) {
	ten := make([]string, MaxMentions)
	for i := range ten {
		ten[i] = fmt.Sprintf("u%d", i)
	}
	for _, c := range []struct {
		name  string
		users []string
		ok    bool
	}{
		{"一个", []string{"user"}, true},
		{"上限", ten, true},
		{"空列表", []string{}, false},
		{"超上限", append(append([]string(nil), ten...), "more"), false},
		{"重复", []string{"a", "a"}, false},
		{"空 ID", []string{""}, false},
		{"ID 太长", []string{strings.Repeat("a", MaxMentionIDBytes+1)}, false},
	} {
		s := validSnapshot()
		s.Mentions = &Mentions{UserIDs: c.users}
		if err := ValidatePublicSnapshot(s); (err == nil) != c.ok {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
}

func TestAttachmentIMFile(t *testing.T) {
	str := func(v string) *string { return &v }
	num := func(v int32) *int32 { return &v }
	with := func(a *AttachmentBlock) *AssistantElem {
		s := validSnapshot()
		a.ResourceID, a.ContentVersion = "res", "v1"
		s.Blocks = append(s.Blocks, &Block{BlockID: "file", Content: &Block_Attachment{Attachment: a}})
		return s
	}
	// 没设时按 EmitUnpopulated 也不输出
	content, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(&AttachmentBlock{ResourceID: "res"})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"kind", "imObject", "width", "height"} {
		if bytes.Contains(content, []byte(field)) {
			t.Fatalf("没设的 %s 不应输出: %s", field, content)
		}
	}
	for _, c := range []struct {
		name string
		a    *AttachmentBlock
		ok   bool
	}{
		{"都不带", &AttachmentBlock{}, true},
		{"图片", &AttachmentBlock{Kind: str("image"), ImObject: str("bot_1/msg_assistant_ab12.png"), Width: num(640), Height: num(480)}, true},
		{"文件", &AttachmentBlock{Kind: str("file"), ImObject: str("bot_1/msg_assistant_ab12.pdf")}, true},
		{"视频没扩展名", &AttachmentBlock{Kind: str("video"), ImObject: str("bot_1/msg_assistant_ab12")}, true},
		{"只有类别", &AttachmentBlock{Kind: str("file")}, false},
		{"只有对象名", &AttachmentBlock{ImObject: str("bot_1/a.pdf")}, false},
		{"只有宽高", &AttachmentBlock{Width: num(1), Height: num(1)}, false},
		{"类别未知", &AttachmentBlock{Kind: str("doc"), ImObject: str("bot_1/a.pdf")}, false},
		{"图片缺宽高", &AttachmentBlock{Kind: str("image"), ImObject: str("bot_1/a.png")}, false},
		{"图片宽为 0", &AttachmentBlock{Kind: str("image"), ImObject: str("bot_1/a.png"), Width: num(0), Height: num(1)}, false},
		{"文件带宽高", &AttachmentBlock{Kind: str("file"), ImObject: str("bot_1/a.pdf"), Width: num(1), Height: num(1)}, false},
		{"空对象名", &AttachmentBlock{Kind: str("file"), ImObject: str("")}, false},
		{"绝对路径", &AttachmentBlock{Kind: str("file"), ImObject: str("/bot_1/a.pdf")}, false},
		{"上级目录", &AttachmentBlock{Kind: str("file"), ImObject: str("../a.pdf")}, false},
		{"空段", &AttachmentBlock{Kind: str("file"), ImObject: str("bot_1//a.pdf")}, false},
		{"空格", &AttachmentBlock{Kind: str("file"), ImObject: str("bot_1/a b.pdf")}, false},
		{"中文", &AttachmentBlock{Kind: str("file"), ImObject: str("bot_1/周报.pdf")}, false},
		{"问号", &AttachmentBlock{Kind: str("file"), ImObject: str("bot_1/a.pdf?x=1")}, false},
		{"太长", &AttachmentBlock{Kind: str("file"), ImObject: str(strings.Repeat("a", MaxIMObjectBytes+1))}, false},
	} {
		if err := ValidatePublicSnapshot(with(c.a)); (err == nil) != c.ok {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
	s := with(&AttachmentBlock{Kind: str("image"), ImObject: str("bot_1/a.png"), Width: num(2), Height: num(3)})
	data, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(s)
	var decoded AssistantElem
	if err != nil || protojson.Unmarshal(data, &decoded) != nil || !proto.Equal(s, &decoded) || !bytes.Contains(data, []byte(`"imObject":"bot_1/a.png"`)) {
		t.Fatalf("IM 文件字段序列化不一致: %s %v", data, err)
	}
}

func TestDocCardBlock(t *testing.T) {
	card := func(id, tid, docType, title string) *Block {
		return &Block{BlockID: id, Content: &Block_DocCard{DocCard: &DocCardBlock{Tid: tid, DocType: docType, Title: title}}}
	}
	s := validSnapshot()
	s.Blocks = append(s.Blocks, card("doc", "t_1-A", "doc", "周报"), card("sheet", "t2", "excel", "销售表"), card("board", "t3", "board", "白板"))
	if err := ValidatePublicSnapshot(s); err != nil {
		t.Fatal(err)
	}
	data, err := protojson.Marshal(s)
	var decoded AssistantElem
	if err != nil || protojson.Unmarshal(data, &decoded) != nil || !proto.Equal(s, &decoded) {
		t.Fatal("文档卡片序列化不一致", err)
	}
	long := ""
	for range MaxDocCardTitleRunes + 1 {
		long += "字"
	}
	for name, b := range map[string]*Block{
		"空载荷":   {BlockID: "x", Content: &Block_DocCard{}},
		"缺编号":   card("x", "", "doc", "周报"),
		"编号带斜杠": card("x", "a/b", "doc", "周报"),
		"编号过长":  card("x", "a234567890123456789012345678901234567890123456789012345678901234x", "doc", "周报"),
		"类型未知":  card("x", "t", "pdf", "周报"),
		"标题空白":  card("x", "t", "doc", "  "),
		"标题过长":  card("x", "t", "doc", long),
		"标题换行":  card("x", "t", "doc", "周\n报"),
		"标题非法":  card("x", "t", "doc", string([]byte{0xff})),
	} {
		bad := validSnapshot()
		bad.Blocks = append(bad.Blocks, b)
		if ValidatePublicSnapshot(bad) == nil {
			t.Fatal("无效文档卡片未被拒绝", name)
		}
	}
	many := validSnapshot()
	for i := range MaxDocCards + 1 {
		many.Blocks = append(many.Blocks, card(fmt.Sprint("doc", i), "t", "doc", "周报"))
	}
	if ValidatePublicSnapshot(many) == nil {
		t.Fatal("文档卡片超过上限未被拒绝")
	}
	many.Blocks = many.Blocks[:len(many.Blocks)-1]
	if err := ValidatePublicSnapshot(many); err != nil {
		t.Fatal("上限以内应通过", err)
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
	for _, text := range []string{"读取附件", "销售表.xlsx · Sheet1", "查询日程 · 3 条", "Read file report.pdf", "Token: 10", "tokens: 1200", "token=abc", "QuarterlySalesReportForEastRegionSummary2026.xlsx", "quarterly_sales_report_for_the_east_region_2026_final_v2.xlsx", "版本 1.2.3", "第 3/5 步"} {
		if UnsafePublicText(text) {
			t.Fatalf("误判: %s", text)
		}
	}
	for _, text := range []string{"/Users/alice/report.pdf", `C:\work\a.txt`, "~/notes.md", "https://example.com/a", "token=ghsdfh28sdf8", "sk-abcdefghijklmnopqrstuv", "rm -rf build", `\\server\share\a`, "打开/Users/alice/a.txt", "读取/data/app/config.yml", "./scripts/run.sh", "密码：123", "api key: abc", "访问 example.com 查询", "连接 10.0.0.12:8080", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig", "d41d8cd98f00b204e9800998ecf8427e", "执行rm -rf /"} {
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
