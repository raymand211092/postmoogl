package smtp

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/emersion/go-smtp"
	"github.com/getsentry/sentry-go"
	"github.com/jhillyerd/enmime"
	"gitlab.com/etke.cc/go/logger"
	"gitlab.com/etke.cc/go/validator"
	"maunium.net/go/mautrix/id"

	"gitlab.com/etke.cc/postmoogle/utils"
)

// incomingSession represents an SMTP-submission session receiving emails from remote servers
type incomingSession struct {
	log          *logger.Logger
	getRoomID    func(string) (id.RoomID, bool)
	getFilters   func(id.RoomID) utils.IncomingFilteringOptions
	receiveEmail func(context.Context, *utils.Email) error
	greylisted   func(net.Addr) bool
	ban          func(net.Addr)
	domains      []string

	ctx  context.Context
	addr net.Addr
	tos  []string
	from string
}

func (s *incomingSession) Mail(from string, opts smtp.MailOptions) error {
	sentry.GetHubFromContext(s.ctx).Scope().SetTag("from", from)
	if !utils.AddressValid(from) {
		s.log.Debug("address %s is invalid", from)
		s.ban(s.addr)
		return ErrBanned
	}
	s.from = from
	s.log.Debug("mail from %s, options: %+v", from, opts)
	return nil
}

func (s *incomingSession) Rcpt(to string) error {
	sentry.GetHubFromContext(s.ctx).Scope().SetTag("to", to)
	s.tos = append(s.tos, to)
	var domainok bool
	for _, domain := range s.domains {
		if utils.Hostname(to) == domain {
			domainok = true
			break
		}
	}
	if !domainok {
		s.log.Debug("wrong domain of %s", to)
		return ErrNoUser
	}

	roomID, ok := s.getRoomID(utils.Mailbox(to))
	if !ok {
		s.log.Debug("mapping for %s not found", to)
		return ErrNoUser
	}

	validations := s.getFilters(roomID)
	if !validateEmail(s.from, to, s.log, validations) {
		s.ban(s.addr)
		return ErrBanned
	}

	s.log.Debug("mail to %s", to)
	return nil
}

func (s *incomingSession) Data(r io.Reader) error {
	if s.greylisted(s.addr) {
		return &smtp.SMTPError{
			Code:         451,
			EnhancedCode: smtp.EnhancedCode{4, 5, 1},
			Message:      "You have been greylisted, try again a bit later.",
		}
	}
	parser := enmime.NewParser()
	eml, err := parser.ReadEnvelope(r)
	if err != nil {
		return err
	}

	email := utils.FromEnvelope(s.tos[0], eml)
	for _, to := range s.tos {
		email.RcptTo = to
		err := s.receiveEmail(s.ctx, email)
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *incomingSession) Reset()        {}
func (s *incomingSession) Logout() error { return nil }

// outgoingSession represents an SMTP-submission session sending emails from external scripts, using postmoogle as SMTP server
type outgoingSession struct {
	log      *logger.Logger
	sendmail func(string, string, string) error
	privkey  string
	domains  []string

	ctx  context.Context
	tos  []string
	from string
}

func (s *outgoingSession) Mail(from string, opts smtp.MailOptions) error {
	sentry.GetHubFromContext(s.ctx).Scope().SetTag("from", from)
	if !utils.AddressValid(from) {
		return errors.New("please, provide email address")
	}
	return nil
}

func (s *outgoingSession) Rcpt(to string) error {
	sentry.GetHubFromContext(s.ctx).Scope().SetTag("to", to)
	s.tos = append(s.tos, to)

	s.log.Debug("mail to %s", to)
	return nil
}

func (s *outgoingSession) Data(r io.Reader) error {
	parser := enmime.NewParser()
	eml, err := parser.ReadEnvelope(r)
	if err != nil {
		return err
	}
	email := utils.FromEnvelope(s.tos[0], eml)
	for _, to := range s.tos {
		email.RcptTo = to
		err := s.sendmail(email.From, to, email.Compose(s.privkey))
		if err != nil {
			return err
		}
	}

	return nil
}
func (s *outgoingSession) Reset()        {}
func (s *outgoingSession) Logout() error { return nil }

func validateEmail(from, to string, log *logger.Logger, options utils.IncomingFilteringOptions) bool {
	enforce := validator.Enforce{
		Email: true,
		MX:    options.SpamcheckMX(),
		SMTP:  options.SpamcheckSMTP(),
	}
	v := validator.New(options.Spamlist(), enforce, to, log)

	return v.Email(from)
}
