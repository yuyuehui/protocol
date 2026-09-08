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

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

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

func TestEmailAccountReqStringRedactsPassword(t *testing.T) {
	req := &TestEmailAccountReq{
		EmailAddress: "user@example.com",
		AuthUser:     "user@example.com",
		AuthPassword: "secret-authorization-code",
	}

	got := req.String()
	if strings.Contains(got, req.AuthPassword) {
		t.Fatalf("String() leaked auth password: %s", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("String() did not include redaction marker: %s", got)
	}
}

func TestEmailUserSettingsSettingsVersionProtoJSONPresence(t *testing.T) {
	withoutVersion, err := protojson.Marshal(&EmailUserSettings{})
	if err != nil {
		t.Fatalf("marshal settings without version: %v", err)
	}
	if strings.Contains(string(withoutVersion), "settingsVersion") {
		t.Fatalf("unset settingsVersion must be omitted, got %s", withoutVersion)
	}

	version := int32(0)
	withVersion, err := protojson.Marshal(&EmailUserSettings{SettingsVersion: &version})
	if err != nil {
		t.Fatalf("marshal settings with version: %v", err)
	}
	if !strings.Contains(string(withVersion), `"settingsVersion":0`) {
		t.Fatalf("explicit zero settingsVersion must be present, got %s", withVersion)
	}
}

func TestUpdateEmailUserSettingsReqPreservesFieldPresence(t *testing.T) {
	clearSignatures := true
	empty := ""
	fontSize := int32(0)
	settingsVersion := int32(1)
	request := &UpdateEmailUserSettingsReq{
		UserID:               "user-1",
		DefaultSenderAddress: &empty,
		FontStyle:            &UpdateEmailFontStyle{FontSize: &fontSize},
		Signatures:           []*EmailSignature{},
		UpdateSignatures:     &clearSignatures,
		SettingsVersion:      &settingsVersion,
	}

	data, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("marshal settings update: %v", err)
	}
	var decoded UpdateEmailUserSettingsReq
	if err := proto.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal settings update: %v", err)
	}
	if decoded.DefaultSenderAddress == nil || decoded.GetDefaultSenderAddress() != "" {
		t.Fatalf("empty default sender presence was not preserved: %#v", decoded.DefaultSenderAddress)
	}
	if decoded.FontStyle == nil || decoded.FontStyle.FontSize == nil || decoded.FontStyle.GetFontSize() != 0 {
		t.Fatalf("zero font size presence was not preserved: %#v", decoded.FontStyle)
	}
	if decoded.UpdateSignatures == nil || !decoded.GetUpdateSignatures() || len(decoded.Signatures) != 0 {
		t.Fatalf("signature clear intent was not preserved: %#v", &decoded)
	}
	if decoded.SettingsVersion == nil || decoded.GetSettingsVersion() != 1 {
		t.Fatalf("settings version presence was not preserved: %#v", decoded.SettingsVersion)
	}
}
