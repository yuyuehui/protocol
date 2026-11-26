package relation

import "testing"

func TestFormatNilAndEmptyResponses(t *testing.T) {
	for _, test := range []struct {
		name       string
		nilValue   interface{ Format() any }
		emptyValue interface{ Format() any }
	}{
		{"GetPaginationFriendsApplyToResp", (*GetPaginationFriendsApplyToResp)(nil), &GetPaginationFriendsApplyToResp{}},
		{"GetPaginationFriendsApplyFromResp", (*GetPaginationFriendsApplyFromResp)(nil), &GetPaginationFriendsApplyFromResp{}},
		{"GetPaginationFriendsResp", (*GetPaginationFriendsResp)(nil), &GetPaginationFriendsResp{}},
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
