package smtp

import (
	"context"

	"github.com/emersion/go-smtp"
	"github.com/getsentry/sentry-go"
	"gitlab.com/etke.cc/go/logger"

	"gitlab.com/etke.cc/postmoogle/email"
)

var (
	// ErrBanned returned to banned hosts
	ErrBanned = &smtp.SMTPError{
		Code:         554,
		EnhancedCode: smtp.EnhancedCode{5, 5, 4},
		Message:      "please, don't bother me anymore, kupo.",
	}
	// ErrNoUser returned when no such mailbox found
	ErrNoUser = &smtp.SMTPError{
		Code:         550,
		EnhancedCode: smtp.EnhancedCode{5, 5, 0},
		Message:      "no such user here, kupo.",
	}
)

type mailServer struct {
	bot        matrixbot
	log        *logger.Logger
	domains    []string
	mailSender MailSender
}

// Login used for outgoing mail submissions only (when you use postmoogle as smtp server in your scripts)
func (m *mailServer) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	m.log.Debug("Login state=%+v username=%+v", state, username)
	if m.bot.IsBanned(state.RemoteAddr) {
		return nil, ErrBanned
	}

	if !email.AddressValid(username) {
		m.log.Debug("address %s is invalid", username)
		m.bot.Ban(state.RemoteAddr)
		return nil, ErrBanned
	}

	roomID, allow := m.bot.AllowAuth(username, password)
	if !allow {
		m.log.Debug("username=%s or password=<redacted> is invalid", username)
		m.bot.Ban(state.RemoteAddr)
		return nil, ErrBanned
	}

	return &outgoingSession{
		ctx:       sentry.SetHubOnContext(context.Background(), sentry.CurrentHub().Clone()),
		sendmail:  m.SendEmail,
		privkey:   m.bot.GetDKIMprivkey(),
		from:      username,
		log:       m.log,
		domains:   m.domains,
		getRoomID: m.bot.GetMapping,
		fromRoom:  roomID,
		tos:       []string{},
	}, nil
}

// AnonymousLogin used for incoming mail submissions only
func (m *mailServer) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	m.log.Debug("AnonymousLogin state=%+v", state)
	if m.bot.IsBanned(state.RemoteAddr) {
		return nil, ErrBanned
	}

	return &incomingSession{
		ctx:          sentry.SetHubOnContext(context.Background(), sentry.CurrentHub().Clone()),
		getRoomID:    m.bot.GetMapping,
		getFilters:   m.bot.GetIFOptions,
		receiveEmail: m.ReceiveEmail,
		ban:          m.bot.Ban,
		greylisted:   m.bot.IsGreylisted,
		trusted:      m.bot.IsTrusted,
		log:          m.log,
		domains:      m.domains,
		addr:         state.RemoteAddr,
		tos:          []string{},
	}, nil
}

// SendEmail to external mail server
func (m *mailServer) SendEmail(from, to, data string) error {
	return m.mailSender.Send(from, to, data)
}

// ReceiveEmail - incoming mail into matrix room
func (m *mailServer) ReceiveEmail(ctx context.Context, eml *email.Email) error {
	return m.bot.IncomingEmail(ctx, eml)
}
