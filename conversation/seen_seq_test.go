package conversation

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestSeenSeqProtocol(t *testing.T) {
	field := (&Conversation{}).ProtoReflect().Descriptor().Fields().ByName("seenSeq")
	if field == nil || field.Number() != protoreflect.FieldNumber(24) {
		t.Fatalf("conversation seenSeq must use field 24, got %v", field)
	}

	method := File_conversation_conversation_proto.Services().ByName("conversation").Methods().ByName("SetConversationSeenSeq")
	if method == nil || method.Input().FullName() != "openim.conversation.SetConversationSeenSeqReq" || method.Output().FullName() != "openim.conversation.SetConversationSeenSeqResp" {
		t.Fatalf("unexpected SetConversationSeenSeq RPC descriptor: %v", method)
	}

	for _, tc := range []struct {
		name string
		msg  proto.Message
	}{
		{"conversation", &Conversation{OwnerUserID: "user", ConversationID: "conversation", SeenSeq: 12}},
		{"request", &SetConversationSeenSeqReq{UserID: "user", ConversationID: "conversation", SeenSeq: 12}},
		{"response", &SetConversationSeenSeqResp{SeenSeq: 12, UnreadCount: 3}},

	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := proto.Marshal(tc.msg)
			if err != nil {
				t.Fatal(err)
			}
			got := proto.Clone(tc.msg)
			proto.Reset(got)
			if err := proto.Unmarshal(data, got); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(got, tc.msg) {
				t.Fatalf("round trip mismatch: got %v, want %v", got, tc.msg)
			}
		})
	}
}
