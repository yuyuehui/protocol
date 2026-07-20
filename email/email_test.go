// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package email

import "testing"

func TestSendEmailReqCheckRejectsInvalidRecipientsAndHeaders(t *testing.T) {
	tests := []struct {
		name string
		req  *SendEmailReq
	}{
		{
			name: "header injection",
			req: &SendEmailReq{
				AccountID: "account-1",
				To:        []*EmailAddress{{Address: "recipient@example.com"}},
				Subject:   "hello\r\nBcc: attacker@example.com",
			},
		},
		{
			name: "invalid recipient",
			req: &SendEmailReq{
				AccountID: "account-1",
				To:        []*EmailAddress{{Address: "not-an-email"}},
				Subject:   "hello",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.req.Check(); err == nil {
				t.Fatal("invalid request was accepted")
			}
		})
	}
}
