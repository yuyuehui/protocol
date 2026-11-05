package agent

import "errors"

var (
	errAgentIDEmpty   = errors.New("agentID is empty")
	errRequestIDEmpty = errors.New("requestID is empty")
	errScopeInvalid   = errors.New("scope is invalid")
	errUserIDEmpty    = errors.New("userID is empty")
	errReviewIDEmpty  = errors.New("reviewID is empty")
	errOperatorEmpty  = errors.New("operatorUserID is empty")
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
	case AgentOrigin_AGENT_ORIGIN_SERVICE:
		if x.ServiceID == "" {
			return errors.New("serviceID is empty")
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

func (x *UpdateAvatarReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
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

func (x *ConfirmKnowledgeReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if len(x.Items) == 0 {
		return errors.New("items is empty")
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

func (x *TestAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.RequestID == "" {
		return errRequestIDEmpty
	}
	if x.Question == "" {
		return errors.New("question is empty")
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

func (x *SelfUseReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
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

func (x *DisableReleaseReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.Scope != ReleaseScope_RELEASE_SCOPE_SELF && x.Scope != ReleaseScope_RELEASE_SCOPE_COMPANY {
		return errScopeInvalid
	}
	if x.Reason == "" {
		return errors.New("reason is empty")
	}
	return nil
}

func (x *EnableReleaseReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.Scope != ReleaseScope_RELEASE_SCOPE_SELF && x.Scope != ReleaseScope_RELEASE_SCOPE_COMPANY {
		return errScopeInvalid
	}
	return nil
}

func (x *SubmitCheckReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *ListGrantsReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}

func (x *SetEditorReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.UserID == "" {
		return errUserIDEmpty
	}
	return nil
}

func (x *RemoveEditorReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.UserID == "" {
		return errUserIDEmpty
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
	if x.Scope != ReleaseScope_RELEASE_SCOPE_UNSPECIFIED && x.Scope != ReleaseScope_RELEASE_SCOPE_COMPANY && x.Scope != ReleaseScope_RELEASE_SCOPE_PLATFORM {
		return errScopeInvalid
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

func (x *RestrictReleaseReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.Scope != ReleaseScope_RELEASE_SCOPE_COMPANY {
		return errScopeInvalid
	}
	if x.RequestID == "" {
		return errRequestIDEmpty
	}
	return nil
}

func (x *ChatReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.UserID == "" {
		return errUserIDEmpty
	}
	if x.ConversationID == "" {
		return errors.New("conversationID is empty")
	}
	if x.RequestID == "" {
		return errRequestIDEmpty
	}
	if x.Question == "" {
		return errors.New("question is empty")
	}
	return nil
}

func (x *AdminGetCompanySettingReq) Check() error {
	if x.CompanyID == "" {
		return errors.New("companyID is empty")
	}
	return nil
}

func (x *AdminSetCompanySettingReq) Check() error {
	if x.CompanyID == "" {
		return errors.New("companyID is empty")
	}
	return nil
}

func (x *AdminSaveModelReq) Check() error {
	if x.Model == nil {
		return errors.New("model is empty")
	}
	return nil
}

func (x *AdminDeleteModelReq) Check() error {
	if x.ModelID == "" {
		return errors.New("modelID is empty")
	}
	return nil
}

func (x *AdminTestModelReq) Check() error {
	if x.ModelID == "" {
		return errors.New("modelID is empty")
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
	if x.Scope < ReleaseScope_RELEASE_SCOPE_SELF || x.Scope > ReleaseScope_RELEASE_SCOPE_PLATFORM {
		return errScopeInvalid
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
	if x.Scope < ReleaseScope_RELEASE_SCOPE_SELF || x.Scope > ReleaseScope_RELEASE_SCOPE_PLATFORM {
		return errScopeInvalid
	}
	return nil
}

func (x *AdminScopeDisableAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.Scope != ReleaseScope_RELEASE_SCOPE_COMPANY {
		return errScopeInvalid
	}
	if x.OperatorUserID == "" {
		return errOperatorEmpty
	}
	if x.Reason == "" {
		return errors.New("reason is empty")
	}
	return nil
}

func (x *AdminScopeEnableAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	if x.Scope != ReleaseScope_RELEASE_SCOPE_COMPANY {
		return errScopeInvalid
	}
	if x.OperatorUserID == "" {
		return errOperatorEmpty
	}
	return nil
}

func (x *AdminRestoreAgentReq) Check() error {
	if x.AgentID == "" {
		return errAgentIDEmpty
	}
	return nil
}
