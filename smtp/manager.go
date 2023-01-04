package smtp

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/emersion/go-smtp"
	"gitlab.com/etke.cc/go/logger"
	"maunium.net/go/mautrix/id"

	"gitlab.com/etke.cc/postmoogle/email"
)

type Config struct {
	Domains []string
	Port    string

	TLSCerts    []string
	TLSKeys     []string
	TLSPort     string
	TLSRequired bool

	LogLevel string
	MaxSize  int
	Bot      matrixbot
	Callers  []Caller
}

type Manager struct {
	log  *logger.Logger
	bot  matrixbot
	smtp *smtp.Server
	errs chan error

	port    string
	tlsPort string
	tlsCfg  *tls.Config
}

type matrixbot interface {
	AllowAuth(string, string) (id.RoomID, bool)
	IsGreylisted(net.Addr) bool
	IsBanned(net.Addr) bool
	IsTrusted(net.Addr) bool
	Ban(net.Addr)
	GetMapping(string) (id.RoomID, bool)
	GetIFOptions(id.RoomID) email.IncomingFilteringOptions
	IncomingEmail(context.Context, *email.Email) error
	GetDKIMprivkey() string
}

// Caller is Sendmail caller
type Caller interface {
	SetSendmail(func(string, string, string) error)
}

// NewManager creates new SMTP server manager
func NewManager(cfg *Config) *Manager {
	log := logger.New("smtp.", cfg.LogLevel)
	mailsrv := &mailServer{
		log:     log,
		bot:     cfg.Bot,
		domains: cfg.Domains,
	}
	for _, caller := range cfg.Callers {
		caller.SetSendmail(mailsrv.SendEmail)
	}

	s := smtp.NewServer(mailsrv)
	s.ErrorLog = loggerWrapper{func(s string, i ...interface{}) { log.Error(s, i...) }}
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = cfg.MaxSize * 1024 * 1024
	s.AllowInsecureAuth = !cfg.TLSRequired
	s.EnableREQUIRETLS = cfg.TLSRequired
	s.EnableSMTPUTF8 = true
	// set domain in greeting only in single-domain mode
	if len(cfg.Domains) == 1 {
		s.Domain = cfg.Domains[0]
	}
	if log.GetLevel() == "INFO" || log.GetLevel() == "DEBUG" || log.GetLevel() == "TRACE" {
		s.Debug = loggerWriter{func(s string) { log.Info(s) }}
	}

	m := &Manager{
		smtp:    s,
		bot:     cfg.Bot,
		log:     log,
		port:    cfg.Port,
		tlsPort: cfg.TLSPort,
	}
	m.loadTLSConfig(cfg.TLSCerts, cfg.TLSKeys)
	return m
}

// Start SMTP server
func (m *Manager) Start() error {
	m.errs = make(chan error, 1)
	go m.listen(m.port, nil)
	if m.tlsCfg != nil {
		go m.listen(m.tlsPort, m.tlsCfg)
	}

	return <-m.errs
}

// Stop SMTP server
func (m *Manager) Stop() {
	err := m.smtp.Close()
	if err != nil {
		m.log.Error("cannot stop SMTP server properly: %v", err)
	}
	m.log.Info("SMTP server has been stopped")
}

func (m *Manager) listen(port string, tlsCfg *tls.Config) {
	var l net.Listener
	var err error
	if tlsCfg != nil {
		l, err = tls.Listen("tcp", ":"+port, tlsCfg)
	} else {
		l, err = net.Listen("tcp", ":"+port)
	}
	if err != nil {
		m.log.Error("cannot start listener on %s: %v", port, err)
		m.errs <- err
		return
	}
	lwrapper := NewListener(l, m.bot.IsBanned, m.log)
	m.log.Info("Starting SMTP server on port %s", port)

	err = m.smtp.Serve(lwrapper)
	if err != nil {
		m.log.Error("cannot start SMTP server on %s: %v", port, err)
		m.errs <- err
		close(m.errs)
	}
}

func (m *Manager) loadTLSConfig(certs, keys []string) {
	if len(certs) == 0 || len(keys) == 0 {
		m.log.Warn("SSL certificates are not provided")
		return
	}

	certificates := make([]tls.Certificate, 0, len(certs))
	for i, path := range certs {
		tlsCert, err := tls.LoadX509KeyPair(path, keys[i])
		if err != nil {
			m.log.Error("cannot load SSL certificate: %v", err)
			continue
		}
		certificates = append(certificates, tlsCert)
	}
	if len(certificates) == 0 {
		return
	}

	m.tlsCfg = &tls.Config{Certificates: certificates}
	m.smtp.TLSConfig = m.tlsCfg
}
