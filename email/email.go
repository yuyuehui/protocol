package email

import (
	"net/mail"

	"github.com/openimsdk/tools/errs"
)

const maxEmailRecipients = 100

func validateEmailAddresses(label string, addresses []*EmailAddress) error {
	if len(addresses) > maxEmailRecipients {
		return errs.ErrArgs.WrapMsg(label + " has too many recipients")
	}
	for _, address := range addresses {
		if address == nil || address.Address == "" {
			return errs.ErrArgs.WrapMsg(label + " contains an empty email address")
		}
		if containsHeaderControl(address.Name) || containsHeaderControl(address.Address) {
			return errs.ErrArgs.WrapMsg(label + " contains invalid header characters")
		}
		parsed, err := mail.ParseAddress(address.Address)
		if err != nil || parsed.Address != address.Address {
			return errs.ErrArgs.WrapMsg(label + " contains an invalid email address")
		}
	}
	return nil
}

func containsHeaderControl(value string) bool {
	for _, r := range value {
		if r == '\r' || r == '\n' || r == 0 {
			return true
		}
	}
	return false
}

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
	if containsHeaderControl(x.Subject) || containsHeaderControl(x.InReplyTo) {
		return errs.ErrArgs.WrapMsg("email header contains invalid characters")
	}
	if err := validateEmailAddresses("to", x.To); err != nil {
		return err
	}
	if err := validateEmailAddresses("cc", x.Cc); err != nil {
		return err
	}
	if err := validateEmailAddresses("bcc", x.Bcc); err != nil {
		return err
	}
	for _, reference := range x.References {
		if containsHeaderControl(reference) {
			return errs.ErrArgs.WrapMsg("references contains invalid header characters")
		}
	}
	return nil
}
