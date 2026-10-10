package schedule

import "testing"

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
