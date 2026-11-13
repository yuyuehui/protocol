package agent

import "errors"

var (
	errAgentIDEmpty    = errors.New("agentID is empty")
	errRequestIDEmpty  = errors.New("requestID is empty")
	errUserIDEmpty     = errors.New("userID is empty")
	errReviewIDEmpty   = errors.New("reviewID is empty")
	errGroupIDEmpty    = errors.New("groupID is empty")
	errUserIDsEmpty    = errors.New("userIDs is empty")
	errInstanceIDEmpty = errors.New("instanceID is empty")
)

func (x *CreateAgentReq) Check() error {
	if x.RequestID == "" {
		return errRequestIDEmpty
	}
	switch x.Origin {
	case AgentOrigin_AGENT_ORIGIN_BLANK:
	case AgentOrigin_AGENT_ORIGIN_TEMPLATE:
		if x.TemplateID == "" {
			return errors.New("templateID is empty")
		}
	case AgentOrigin_AGENT_ORIGIN_EXPERT:
		if x.ExpertID == "" {
			return errors.New("expertID is empty")
		}
	default:
		return errors.New("origin is invalid")
	}
	return nil
}

func (x *GetAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *SaveDraftReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.Config == nil {
		return errors.New("config is empty")
	}
	return nil
}

func (x *RestoreDraftReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.SnapshotID == "" {
		return errors.New("snapshotID is empty")
	}
	return nil
}

func (x *DeleteAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *RestoreAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *PreviewRoleImportReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.ObjectID == "" {
		return errors.New("objectID is empty")
	}
	return nil
}

func (x *ApplyRoleImportReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.ObjectID == "" {
		return errors.New("objectID is empty")
	}
	if x.Mode != ImportMode_IMPORT_MODE_REPLACE && x.Mode != ImportMode_IMPORT_MODE_APPEND {
		return errors.New("mode is invalid")
	}
	if x.Zone < ImportZone_IMPORT_ZONE_GOAL || x.Zone > ImportZone_IMPORT_ZONE_OUTPUT {
		return errors.New("zone is invalid")
	}
	return nil
}

func (x *ListTestRunsReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *MarkTestRunReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.RunID == "" {
		return errors.New("runID is empty")
	}
	if x.Verdict != TestVerdict_TEST_VERDICT_MATCH && x.Verdict != TestVerdict_TEST_VERDICT_MISMATCH {
		return errors.New("verdict is invalid")
	}
	return nil
}

func (x *ReportTestRunReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.RequestID == "" {
		return errRequestIDEmpty
	}
	switch x.Result {
	case TestResult_TEST_RESULT_COMPLETED, TestResult_TEST_RESULT_FAILED, TestResult_TEST_RESULT_UNKNOWN:
	default:
		return errors.New("result is invalid")
	}
	return nil
}

func (x *ListReleasesReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *GetSnapshotReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.SnapshotID == "" {
		return errors.New("snapshotID is empty")
	}
	return nil
}

func (x *SubmitCheckReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *ListMembersReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *SetMemberReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.UserID == "" {
		return errUserIDEmpty
	}
	if x.Role != MemberRole_MEMBER_ROLE_OWNER && x.Role != MemberRole_MEMBER_ROLE_COLLABORATOR {
		return errors.New("role is invalid")
	}
	return nil
}

func (x *RemoveMemberReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.UserID == "" {
		return errUserIDEmpty
	}
	return nil
}

func (x *SetVisibleScopeReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.Scope == nil {
		return errors.New("scope is empty")
	}
	return nil
}

func (x *GetTemplateReq) Check() error {
	if x.TemplateID == "" {
		return errors.New("templateID is empty")
	}
	return nil
}

func (x *GetEligibilityReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *ListVersionsReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *DiffVersionReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *SubmitReviewReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.RequestID == "" {
		return errRequestIDEmpty
	}
	return nil
}

func (x *WithdrawReviewReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.ReviewID == "" {
		return errReviewIDEmpty
	}
	return nil
}

func (x *ListReviewsReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *GetReviewReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.ReviewID == "" {
		return errReviewIDEmpty
	}
	return nil
}

func (x *RetryReviewReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.ReviewID == "" {
		return errReviewIDEmpty
	}
	return nil
}

func (x *GetInstanceConfigReq) Check() error {
	if x.InstanceID == "" {
		return errInstanceIDEmpty
	}
	return nil
}

func (x *PrepareInstanceReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.HirerID == "" {
		return errors.New("hirerID is empty")
	}
	return nil
}

func (x *BindInstanceReq) Check() error {
	if x.InstanceID == "" {
		return errInstanceIDEmpty
	}
	if x.ImUserID == "" {
		return errors.New("imUserID is empty")
	}
	return nil
}

func (x *RemoveInstanceReq) Check() error {
	if x.InstanceID == "" && (x.AgentID == "" || x.HirerID == "") {
		return errors.New("instanceID or agentID and hirerID is empty")
	}
	return nil
}

func (x *GetInstanceEligibilityReq) Check() error {
	if x.InstanceID == "" && x.ImUserID == "" {
		return errors.New("instanceID or imUserID is empty")
	}
	return nil
}

func (x *AdminSaveTemplateReq) Check() error {
	if x.Template == nil {
		return errors.New("template is empty")
	}
	return nil
}

func (x *AdminDeleteTemplateReq) Check() error {
	if x.TemplateID == "" {
		return errors.New("templateID is empty")
	}
	return nil
}

func (x *AdminSaveTagReq) Check() error {
	if x.Tag == nil {
		return errors.New("tag is empty")
	}
	return nil
}

func (x *AdminDeleteTagReq) Check() error {
	if x.TagID == "" {
		return errors.New("tagID is empty")
	}
	return nil
}

func (x *AdminDisableAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.Reason == "" {
		return errors.New("reason is empty")
	}
	return nil
}

func (x *AdminEnableAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *AdminDelistAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *AdminRelistAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *AdminDeleteAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *AdminRestoreAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *AdminScopeOptionsReq) Check() error {
	if x.Type < ScopeItemType_SCOPE_ITEM_TYPE_DEPARTMENT || x.Type > ScopeItemType_SCOPE_ITEM_TYPE_USER {
		return errors.New("type is invalid")
	}
	return nil
}

func (x *AdminGetPeopleGroupReq) Check() error {
	if x.GroupID == "" {
		return errGroupIDEmpty
	}
	return nil
}

func (x *AdminListPeopleGroupMembersReq) Check() error {
	if x.GroupID == "" {
		return errGroupIDEmpty
	}
	return nil
}

func (x *AdminListUserPeopleGroupsReq) Check() error {
	if x.UserID == "" {
		return errUserIDEmpty
	}
	return nil
}

func (x *AdminSavePeopleGroupReq) Check() error {
	if x.Group == nil {
		return errors.New("group is empty")
	}
	return nil
}

func (x *AdminDeletePeopleGroupReq) Check() error {
	if x.GroupID == "" {
		return errGroupIDEmpty
	}
	return nil
}

func (x *AdminAddPeopleGroupUsersReq) Check() error {
	if x.GroupID == "" {
		return errGroupIDEmpty
	}
	if len(x.UserIDs) == 0 {
		return errUserIDsEmpty
	}
	return nil
}

func (x *AdminRemovePeopleGroupUsersReq) Check() error {
	if x.GroupID == "" {
		return errGroupIDEmpty
	}
	if len(x.UserIDs) == 0 {
		return errUserIDsEmpty
	}
	return nil
}

func (x *AdminGetReviewReq) Check() error {
	if x.ReviewID == "" {
		return errReviewIDEmpty
	}
	return nil
}

func (x *AdminDecideReviewReq) Check() error {
	if x.ReviewID == "" {
		return errReviewIDEmpty
	}
	return nil
}
