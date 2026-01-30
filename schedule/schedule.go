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

package schedule

import (
	"github.com/openimsdk/tools/errs"
)

// ScheduleRepeatInfo Check 重复规则参数校验
func (x *ScheduleRepeatInfo) Check() error {
	if x.RepeatType == "" {
		return nil // 不重复，允许为空
	}

	// 验证重复类型
	validRepeatTypes := map[string]bool{
		"daily":   true,
		"weekly":  true,
		"monthly": true,
		"yearly":  true,
	}
	if !validRepeatTypes[x.RepeatType] {
		return errs.ErrArgs.WrapMsg("repeatType must be one of: daily, weekly, monthly, yearly")
	}

	// 验证间隔
	if x.Interval <= 0 {
		return errs.ErrArgs.WrapMsg("interval must be greater than 0")
	}

	// 验证单位类型
	if x.UnitType == "" {
		return errs.ErrArgs.WrapMsg("unitType is required when repeatType is set")
	}
	validUnitTypes := map[string]bool{
		"day":     true,
		"weekday": true,
		"week":    true,
		"month":   true,
		"year":    true,
	}
	if !validUnitTypes[x.UnitType] {
		return errs.ErrArgs.WrapMsg("unitType must be one of: day, weekday, week, month, year")
	}

	// 验证重复类型和单位类型的匹配
	if x.RepeatType == "daily" && x.UnitType != "day" && x.UnitType != "weekday" {
		return errs.ErrArgs.WrapMsg("when repeatType is daily, unitType must be day or weekday")
	}
	if x.RepeatType == "weekly" && x.UnitType != "week" {
		return errs.ErrArgs.WrapMsg("when repeatType is weekly, unitType must be week")
	}
	if x.RepeatType == "monthly" && x.UnitType != "month" {
		return errs.ErrArgs.WrapMsg("when repeatType is monthly, unitType must be month")
	}
	if x.RepeatType == "yearly" && x.UnitType != "year" {
		return errs.ErrArgs.WrapMsg("when repeatType is yearly, unitType must be year")
	}

	// 验证每周重复的星期几
	// 当 repeatType 为 weekly 时，必须指定 repeatDaysOfWeek
	if x.RepeatType == "weekly" && len(x.RepeatDaysOfWeek) == 0 {
		return errs.ErrArgs.WrapMsg("repeatDaysOfWeek is required when repeatType is weekly")
	}
	// 当 unitType 为 weekday 时，应该指定工作日（周一到周五）
	if x.UnitType == "weekday" && len(x.RepeatDaysOfWeek) == 0 {
		// 允许不指定，系统会自动使用周一到周五
	}
	// 验证 repeatDaysOfWeek 的值范围（0-6）
	for _, day := range x.RepeatDaysOfWeek {
		if day < 0 || day > 6 {
			return errs.ErrArgs.WrapMsg("repeatDaysOfWeek values must be between 0 (Sunday) and 6 (Saturday)")
		}
	}

	// 验证结束条件：endDate 和 repeatTimes 不能同时设置
	if x.EndDate > 0 && x.RepeatTimes > 0 {
		return errs.ErrArgs.WrapMsg("endDate and repeatTimes cannot be set at the same time")
	}

	// 验证结束日期
	if x.EndDate > 0 {
		// endDate 应该是未来的时间戳
		// 这里不强制验证，因为可能是历史数据
	}

	// 验证重复次数
	if x.RepeatTimes < 0 {
		return errs.ErrArgs.WrapMsg("repeatTimes must be greater than or equal to 0")
	}
	if x.RepeatTimes > 0 && x.RepeatTimes > 1000 {
		return errs.ErrArgs.WrapMsg("repeatTimes cannot exceed 1000")
	}

	return nil
}

// MeetingSettings Check 会议设置参数校验
func (x *MeetingSettings) Check() error {
	// 如果启用了密码，必须提供密码且密码长度在4-6位之间
	if x.EnablePassword {
		if x.Password == "" {
			return errs.ErrArgs.WrapMsg("password is required when enablePassword is true")
		}
		// 验证密码长度（4-6位数字）
		if len(x.Password) < 4 || len(x.Password) > 6 {
			return errs.ErrArgs.WrapMsg("password must be 4-6 digits")
		}
		// 验证密码是否为纯数字
		for _, c := range x.Password {
			if c < '0' || c > '9' {
				return errs.ErrArgs.WrapMsg("password must be numeric")
			}
		}
	}
	// 如果 callReminder 是指定成员，必须提供 callReminderUserIDs
	if x.CallReminder == CallReminderType_CALL_REMINDER_SPECIFIED {
		if len(x.CallReminderUserIDs) == 0 {
			return errs.ErrArgs.WrapMsg("callReminderUserIDs is required when callReminder is SPECIFIED")
		}
	}
	return nil
}

// CreateScheduleReq Check 创建日程请求参数校验
func (x *CreateScheduleReq) Check() error {
	// creatorUserID 可选，如果为空则由服务端从 context 中获取
	if x.Title == "" {
		return errs.ErrArgs.WrapMsg("title is empty")
	}
	if x.StartTime <= 0 {
		return errs.ErrArgs.WrapMsg("startTime is invalid")
	}
	if x.EndTime <= x.StartTime {
		return errs.ErrArgs.WrapMsg("endTime must be greater than startTime")
	}
	// 设置默认类型为日程
	if x.Type == ScheduleType_SCHEDULE || x.Type == 0 {
		x.Type = ScheduleType_SCHEDULE
		// 如果是日程类型，清空会议设置
		if x.MeetingSettings != nil {
			x.MeetingSettings = nil
		}
	} else if x.Type == ScheduleType_MEETING {
		// 如果是会议类型，必须提供会议设置
		if x.MeetingSettings == nil {
			return errs.ErrArgs.WrapMsg("meetingSettings is required when type is MEETING")
		}
		// 验证会议设置
		if err := x.MeetingSettings.Check(); err != nil {
			return err
		}
	} else {
		return errs.ErrArgs.WrapMsg("invalid schedule type")
	}
	// 验证重复规则
	if x.RepeatInfo != nil {
		if err := x.RepeatInfo.Check(); err != nil {
			return err
		}
	}
	return nil
}

// UpdateScheduleReq Check 更新日程请求参数校验
func (x *UpdateScheduleReq) Check() error {
	if x.ScheduleID == "" {
		return errs.ErrArgs.WrapMsg("scheduleID is empty")
	}
	if x.OperatorUserID == "" {
		return errs.ErrArgs.WrapMsg("operatorUserID is empty")
	}
	// 如果提供了时间，需要校验时间有效性
	if x.StartTime != nil && x.EndTime != nil {
		if *x.StartTime <= 0 {
			return errs.ErrArgs.WrapMsg("startTime is invalid")
		}
		if *x.EndTime <= *x.StartTime {
			return errs.ErrArgs.WrapMsg("endTime must be greater than startTime")
		}
	}
	// 如果提供了类型，需要校验类型和会议设置的一致性
	if x.Type != nil {
		scheduleType := ScheduleType(*x.Type)
		if scheduleType == ScheduleType_MEETING {
			// 如果是会议类型，必须提供会议设置
			if x.MeetingSettings == nil {
				return errs.ErrArgs.WrapMsg("meetingSettings is required when type is MEETING")
			}
			// 验证会议设置
			if err := x.MeetingSettings.Check(); err != nil {
				return err
			}
		} else if scheduleType == ScheduleType_SCHEDULE {
			// 如果是日程类型，清空会议设置
			if x.MeetingSettings != nil {
				return errs.ErrArgs.WrapMsg("meetingSettings should be nil when type is SCHEDULE")
			}
		} else {
			return errs.ErrArgs.WrapMsg("invalid schedule type")
		}
	} else if x.MeetingSettings != nil {
		// 如果提供了会议设置但没有提供类型，需要验证会议设置
		if err := x.MeetingSettings.Check(); err != nil {
			return err
		}
	}
	// 验证重复规则
	if x.RepeatInfo != nil {
		if err := x.RepeatInfo.Check(); err != nil {
			return err
		}
	}
	return nil
}

// DeleteScheduleReq Check 删除日程请求参数校验
func (x *DeleteScheduleReq) Check() error {
	if x.ScheduleID == "" {
		return errs.ErrArgs.WrapMsg("scheduleID is empty")
	}
	if x.OperatorUserID == "" {
		return errs.ErrArgs.WrapMsg("operatorUserID is empty")
	}
	return nil
}

// GetScheduleReq Check 查询日程详情请求参数校验
func (x *GetScheduleReq) Check() error {
	if x.ScheduleID == "" {
		return errs.ErrArgs.WrapMsg("scheduleID is empty")
	}
	if x.UserID == "" {
		return errs.ErrArgs.WrapMsg("userID is empty")
	}
	return nil
}

// GetSchedulesReq Check 查询日程列表请求参数校验
func (x *GetSchedulesReq) Check() error {
	// userID 可选，如果不传则使用 context 中的当前用户
	if x.StartTime > 0 && x.EndTime > 0 {
		if x.EndTime <= x.StartTime {
			return errs.ErrArgs.WrapMsg("endTime must be greater than startTime")
		}
	}
	if x.Pagination != nil {
		if x.Pagination.PageNumber < 1 {
			return errs.ErrArgs.WrapMsg("pageNumber is invalid")
		}
		if x.Pagination.ShowNumber < 1 {
			return errs.ErrArgs.WrapMsg("showNumber is invalid")
		}
	}
	return nil
}

// AcceptScheduleReq Check 接受日程邀请请求参数校验
func (x *AcceptScheduleReq) Check() error {
	if x.ScheduleID == "" {
		return errs.ErrArgs.WrapMsg("scheduleID is empty")
	}
	if x.UserID == "" {
		return errs.ErrArgs.WrapMsg("userID is empty")
	}
	return nil
}

// RejectScheduleReq Check 拒绝日程邀请请求参数校验
func (x *RejectScheduleReq) Check() error {
	if x.ScheduleID == "" {
		return errs.ErrArgs.WrapMsg("scheduleID is empty")
	}
	if x.UserID == "" {
		return errs.ErrArgs.WrapMsg("userID is empty")
	}
	return nil
}

// SetReminderReq Check 设置提醒请求参数校验
func (x *SetReminderReq) Check() error {
	if x.ScheduleID == "" {
		return errs.ErrArgs.WrapMsg("scheduleID is empty")
	}
	if x.UserID == "" {
		return errs.ErrArgs.WrapMsg("userID is empty")
	}
	return nil
}

// CheckConflictReq Check 查询日程冲突请求参数校验
func (x *CheckConflictReq) Check() error {
	if len(x.UserIDs) == 0 {
		return errs.ErrArgs.WrapMsg("userIDs is required")
	}
	if x.StartTime <= 0 {
		return errs.ErrArgs.WrapMsg("startTime is invalid")
	}
	if x.EndTime <= x.StartTime {
		return errs.ErrArgs.WrapMsg("endTime must be greater than startTime")
	}
	return nil
}

// GetScheduleDatesReq Check 查询某个月有参与日程的日期列表请求参数校验
func (x *GetScheduleDatesReq) Check() error {
	if x.UserID == "" {
		return errs.ErrArgs.WrapMsg("userID is empty")
	}
	if x.Year <= 0 {
		return errs.ErrArgs.WrapMsg("year is invalid")
	}
	if x.Month < 1 || x.Month > 12 {
		return errs.ErrArgs.WrapMsg("month must be between 1 and 12")
	}
	return nil
}

// GetScheduleMonthViewReq Check 查询某个月每天的所有日程请求参数校验
func (x *GetScheduleMonthViewReq) Check() error {
	if x.UserID == "" {
		return errs.ErrArgs.WrapMsg("userID is empty")
	}
	if x.Year <= 0 {
		return errs.ErrArgs.WrapMsg("year is invalid")
	}
	if x.Month < 1 || x.Month > 12 {
		return errs.ErrArgs.WrapMsg("month must be between 1 and 12")
	}
	return nil
}

// SendScheduleMessageReq Check 发送日程消息到聊天请求参数校验
func (x *SendScheduleMessageReq) Check() error {
	if len(x.ScheduleIDs) == 0 {
		return errs.ErrArgs.WrapMsg("scheduleIDs is empty, at least one scheduleID is required")
	}
	// 发送模式：0=创建日程发送消息（默认），1=转发消息
	// sendMode=0：用 userIDs 创建群聊并发送消息
	// sendMode=1：userIDs 每个人发消息，groupIDs 分别发群消息（可以同时存在）
	sendMode := x.SendMode
	if sendMode == 0 {
		sendMode = 0 // 默认创建日程发送消息模式
	}
	// 创建日程发送消息模式（sendMode=0）：userIDs 必填
	if sendMode == 0 {
		if len(x.UserIDs) == 0 {
			return errs.ErrArgs.WrapMsg("userIDs is required when sendMode is 0 (create group and send message)")
		}
		// 创建日程发送消息模式不支持 groupIDs
		if len(x.GroupIDs) > 0 {
			return errs.ErrArgs.WrapMsg("groupIDs is not supported when sendMode is 0 (create group and send message)")
		}
	} else if sendMode == 1 {
		// 转发模式（sendMode=1）：userIDs 和 groupIDs 可以同时设置
		// 如果同时设置，会先发送到 userIDs（单聊），再发送到 groupIDs（群聊）
	} else {
		return errs.ErrArgs.WrapMsg("invalid sendMode, must be 0 (create group and send message) or 1 (forward message)")
	}
	return nil
}

// InitScheduleGroupsReq Check 初始化日程分组请求参数校验
func (x *InitScheduleGroupsReq) Check() error {
	if x.OwnerUserID == "" {
		return errs.ErrArgs.WrapMsg("ownerUserID is empty")
	}
	return nil
}

// GetAllScheduleGroupsReq Check 获取所有日程分组请求参数校验
func (x *GetAllScheduleGroupsReq) Check() error {
	if x.OwnerUserID == "" {
		return errs.ErrArgs.WrapMsg("ownerUserID is empty")
	}
	return nil
}

// CreateScheduleGroupReq Check 创建日程分组请求参数校验
func (x *CreateScheduleGroupReq) Check() error {
	if x.OwnerUserID == "" {
		return errs.ErrArgs.WrapMsg("ownerUserID is empty")
	}
	if x.GroupName == "" {
		return errs.ErrArgs.WrapMsg("groupName is required")
	}
	return nil
}

// UpdateScheduleGroupReq Check 更新日程分组请求参数校验
func (x *UpdateScheduleGroupReq) Check() error {
	if x.GroupID == "" {
		return errs.ErrArgs.WrapMsg("groupID is required")
	}
	return nil
}

// DeleteScheduleGroupReq Check 删除日程分组请求参数校验
func (x *DeleteScheduleGroupReq) Check() error {
	if x.GroupID == "" {
		return errs.ErrArgs.WrapMsg("groupID is required")
	}
	return nil
}
