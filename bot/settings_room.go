package bot

import (
	"strings"

	"maunium.net/go/mautrix/id"

	"gitlab.com/etke.cc/postmoogle/utils"
)

// account data key
const acRoomSettingsKey = "cc.etke.postmoogle.settings"

// option keys
const (
	roomOptionOwner              = "owner"
	roomOptionMailbox            = "mailbox"
	roomOptionNoSend             = "nosend"
	roomOptionNoSender           = "nosender"
	roomOptionNoRecipient        = "norecipient"
	roomOptionNoSubject          = "nosubject"
	roomOptionNoHTML             = "nohtml"
	roomOptionNoThreads          = "nothreads"
	roomOptionNoFiles            = "nofiles"
	roomOptionPassword           = "password"
	roomOptionSpamcheckSMTP      = "spamcheck:smtp"
	roomOptionSpamcheckMX        = "spamcheck:mx"
	roomOptionSpamlistEmails     = "spamlist:emails"
	roomOptionSpamlistHosts      = "spamlist:hosts"
	roomOptionSpamlistLocalparts = "spamlist:mailboxes"
)

type roomSettings map[string]string

// Get option
func (s roomSettings) Get(key string) string {
	return s[strings.ToLower(strings.TrimSpace(key))]
}

// Set option
func (s roomSettings) Set(key, value string) {
	s[strings.ToLower(strings.TrimSpace(key))] = value
}

func (s roomSettings) Mailbox() string {
	return s.Get(roomOptionMailbox)
}

func (s roomSettings) Owner() string {
	return s.Get(roomOptionOwner)
}

func (s roomSettings) Password() string {
	return s.Get(roomOptionPassword)
}

func (s roomSettings) NoSend() bool {
	return utils.Bool(s.Get(roomOptionNoSend))
}

func (s roomSettings) NoSender() bool {
	return utils.Bool(s.Get(roomOptionNoSender))
}

func (s roomSettings) NoRecipient() bool {
	return utils.Bool(s.Get(roomOptionNoRecipient))
}

func (s roomSettings) NoSubject() bool {
	return utils.Bool(s.Get(roomOptionNoSubject))
}

func (s roomSettings) NoHTML() bool {
	return utils.Bool(s.Get(roomOptionNoHTML))
}

func (s roomSettings) NoThreads() bool {
	return utils.Bool(s.Get(roomOptionNoThreads))
}

func (s roomSettings) NoFiles() bool {
	return utils.Bool(s.Get(roomOptionNoFiles))
}

func (s roomSettings) SpamcheckSMTP() bool {
	return utils.Bool(s.Get(roomOptionSpamcheckSMTP))
}

func (s roomSettings) SpamcheckMX() bool {
	return utils.Bool(s.Get(roomOptionSpamcheckMX))
}

func (s roomSettings) SpamlistEmails() []string {
	return utils.StringSlice(s.Get(roomOptionSpamlistEmails))
}

func (s roomSettings) SpamlistHosts() []string {
	return utils.StringSlice(s.Get(roomOptionSpamlistHosts))
}

func (s roomSettings) SpamlistLocalparts() []string {
	return utils.StringSlice(s.Get(roomOptionSpamlistLocalparts))
}

// ContentOptions converts room display settings to content options
func (s roomSettings) ContentOptions() *utils.ContentOptions {
	return &utils.ContentOptions{
		HTML:      !s.NoHTML(),
		Sender:    !s.NoSender(),
		Recipient: !s.NoRecipient(),
		Subject:   !s.NoSubject(),
		Threads:   !s.NoThreads(),

		FromKey:      eventFromKey,
		SubjectKey:   eventSubjectKey,
		MessageIDKey: eventMessageIDkey,
		InReplyToKey: eventInReplyToKey,
	}
}

func (b *Bot) getRoomSettings(roomID id.RoomID) (roomSettings, error) {
	config, err := b.lp.GetRoomAccountData(roomID, acRoomSettingsKey)
	if config == nil {
		config = map[string]string{}
	}

	return config, utils.UnwrapError(err)
}

func (b *Bot) setRoomSettings(roomID id.RoomID, cfg roomSettings) error {
	return utils.UnwrapError(b.lp.SetRoomAccountData(roomID, acRoomSettingsKey, cfg))
}
