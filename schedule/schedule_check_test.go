package schedule

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestScheduleRepeatInfoCheckYearlyDate(t *testing.T) {
	cases := []struct {
		month, day int32
		ok         bool
	}{
		{2, 29, true},
		{2, 30, false},
		{2, 31, false},
		{4, 31, false},
		{6, 31, false},
		{9, 31, false},
		{11, 31, false},
		{1, 31, true},
		{12, 31, true},
		{4, 30, true},
	}
	for _, c := range cases {
		info := &ScheduleRepeatInfo{RepeatType: "yearly", UnitType: "year", Interval: 1, RepeatMonth: c.month, RepeatDayOfMonth: c.day}
		err := info.Check()
		if c.ok && err != nil {
			t.Fatalf("%d-%d should be valid, got %v", c.month, c.day, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("%d-%d should be invalid", c.month, c.day)
		}
	}
}

func TestUpdateScheduleReqClearFieldsSurviveProtobuf(t *testing.T) {
	var req UpdateScheduleReq
	body := `{"scheduleID":"s1","reminders":[],"attendeeUserIDs":[],"attachmentURLs":null}`
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	req.MarkClearFields()
	req.MarkClearFields()
	if err := req.Check(); err != nil {
		t.Fatalf("check failed: %v", err)
	}
	b, err := proto.Marshal(&req)
	if err != nil {
		t.Fatal(err)
	}
	var got UpdateScheduleReq
	if err := proto.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Reminders != nil || got.AttendeeUserIDs != nil {
		t.Fatal("protobuf should drop empty repeated fields, clearFields is required")
	}
	got.ApplyClearFields()
	if got.Reminders == nil || len(got.Reminders) != 0 {
		t.Fatalf("reminders should be explicit empty, got %#v", got.Reminders)
	}
	if got.AttendeeUserIDs == nil || len(got.AttendeeUserIDs) != 0 {
		t.Fatalf("attendeeUserIDs should be explicit empty, got %#v", got.AttendeeUserIDs)
	}
	if got.AttachmentURLs != nil {
		t.Fatalf("attachmentURLs was not sent and must stay nil, got %#v", got.AttachmentURLs)
	}
	if len(got.ClearFields) != 2 {
		t.Fatalf("clearFields should be deduplicated, got %v", got.ClearFields)
	}
}

func TestUpdateScheduleReqClearFieldsCheck(t *testing.T) {
	conflict := &UpdateScheduleReq{ScheduleID: "s1", Reminders: []int32{1}, ClearFields: []string{ClearFieldReminders}}
	if conflict.Check() == nil {
		t.Fatal("clearing and setting the same field should fail")
	}
	unknown := &UpdateScheduleReq{ScheduleID: "s1", ClearFields: []string{"title"}}
	if unknown.Check() == nil {
		t.Fatal("unknown clear field should fail")
	}
	ok := &UpdateScheduleReq{ScheduleID: "s1", ClearFields: []string{ClearFieldAttachmentURLs}}
	if err := ok.Check(); err != nil {
		t.Fatalf("valid clear field rejected: %v", err)
	}
}
