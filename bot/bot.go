package bot

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/rs/zerolog"
	"gitlab.com/etke.cc/linkpearl"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"gitlab.com/etke.cc/postmoogle/bot/config"
	"gitlab.com/etke.cc/postmoogle/bot/queue"
	"gitlab.com/etke.cc/postmoogle/utils"
)

// Mailboxes config
type MBXConfig struct {
	Reserved   []string
	Forwarded  []string
	Activation string
}

// Bot represents matrix bot
type Bot struct {
	prefix                  string
	mbxc                    MBXConfig
	domains                 []string
	allowedUsers            []*regexp.Regexp
	allowedAdmins           []*regexp.Regexp
	adminRooms              []id.RoomID
	ignoreBefore            int64 // mautrix 0.15.x migration
	commands                commandList
	rooms                   sync.Map
	proxies                 []string
	sendmail                func(string, string, string) error
	cfg                     *config.Manager
	log                     *zerolog.Logger
	lp                      *linkpearl.Linkpearl
	mu                      utils.Mutex
	q                       *queue.Queue
	handledMembershipEvents sync.Map
}

// New creates a new matrix bot
func New(
	q *queue.Queue,
	lp *linkpearl.Linkpearl,
	log *zerolog.Logger,
	cfg *config.Manager,
	proxies []string,
	prefix string,
	domains []string,
	admins []string,
	mbxc MBXConfig,
) (*Bot, error) {
	b := &Bot{
		domains:    domains,
		prefix:     prefix,
		rooms:      sync.Map{},
		adminRooms: []id.RoomID{},
		proxies:    proxies,
		mbxc:       mbxc,
		cfg:        cfg,
		log:        log,
		lp:         lp,
		mu:         utils.NewMutex(),
		q:          q,
	}
	users, err := b.initBotUsers()
	if err != nil {
		return nil, err
	}
	allowedUsers, uerr := parseMXIDpatterns(users, "")
	if uerr != nil {
		return nil, uerr
	}
	b.allowedUsers = allowedUsers

	allowedAdmins, aerr := parseMXIDpatterns(admins, "")
	if aerr != nil {
		return nil, aerr
	}
	b.allowedAdmins = allowedAdmins

	b.commands = b.initCommands()

	return b, nil
}

// Error message to the log and matrix room
func (b *Bot) Error(ctx context.Context, message string, args ...any) {
	evt := eventFromContext(ctx)
	threadID := threadIDFromContext(ctx)
	if threadID == "" {
		threadID = linkpearl.EventParent(evt.ID, evt.Content.AsMessage())
	}

	err := fmt.Errorf(message, args...) //nolint:goerr113 // we have to
	b.log.Error().Err(err).Msg(err.Error())
	if evt == nil {
		return
	}

	var noThreads bool
	cfg, cerr := b.cfg.GetRoom(evt.RoomID)
	if cerr == nil {
		noThreads = cfg.NoThreads()
	}

	var relatesTo *event.RelatesTo
	if threadID != "" {
		relatesTo = linkpearl.RelatesTo(threadID, noThreads)
	}

	b.lp.SendNotice(evt.RoomID, "ERROR: "+err.Error(), relatesTo)
}

// Start performs matrix /sync
func (b *Bot) Start(statusMsg string) error {
	if err := b.migrateMautrix015(); err != nil {
		return err
	}

	if err := b.syncRooms(); err != nil {
		return err
	}

	b.initSync()
	b.log.Info().Msg("Postmoogle has been started")
	return b.lp.Start(statusMsg)
}

// Stop the bot
func (b *Bot) Stop() {
	err := b.lp.GetClient().SetPresence(event.PresenceOffline)
	if err != nil {
		b.log.Error().Err(err).Msg("cannot set presence = offline")
	}
	b.lp.GetClient().StopSync()
}
