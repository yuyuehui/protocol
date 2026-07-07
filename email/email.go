package email

import (
	"github.com/openimsdk/tools/errs"
)

func (x *AddEmailAccountReq) Check() error {
	if x.EmailAddress == "" {
		return errs.ErrArgs.WrapMsg("emailAddress is required")
	}
	if x.ImapHost == "" {
		return errs.ErrArgs.WrapMsg("imapHost is required")
	}
	if x.SmtpHost == "" {
		return errs.ErrArgs.WrapMsg("smtpHost is required")
	}
	if x.AuthUser == "" {
		return errs.ErrArgs.WrapMsg("authUser is required")
	}
	if x.AuthPassword == "" {
		return errs.ErrArgs.WrapMsg("authPassword is required")
	}
	return nil
}

func (x *SendEmailReq) Check() error {
	if x.AccountID == "" {
		return errs.ErrArgs.WrapMsg("accountID is required")
	}
	if len(x.To) == 0 {
		return errs.ErrArgs.WrapMsg("at least one recipient is required")
	}
	if x.Subject == "" && x.TextBody == "" && x.HtmlBody == "" {
		return errs.ErrArgs.WrapMsg("subject or body is required")
	}
	return nil
}
