package conversation

import "testing"

func TestFormatNilAndEmptyResponses(t *testing.T) {
	for _, test := range []struct {
		name       string
		nilValue   interface{ Format() any }
		emptyValue interface{ Format() any }
	}{
		{"GetConversationIDsResp", (*GetConversationIDsResp)(nil), &GetConversationIDsResp{}},
		{"GetOwnerConversationResp", (*GetOwnerConversationResp)(nil), &GetOwnerConversationResp{}},
		{"GetAllConversationsResp", (*GetAllConversationsResp)(nil), &GetAllConversationsResp{}},
		{"GetFullOwnerConversationIDsResp", (*GetFullOwnerConversationIDsResp)(nil), &GetFullOwnerConversationIDsResp{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.nilValue.Format(); got != nil {
				t.Fatalf("nil response formatted as %#v", got)
			}
			if got := test.emptyValue.Format(); got != test.emptyValue {
				t.Fatalf("empty response should retain its identity: %#v", got)
			}
		})
	}
}
