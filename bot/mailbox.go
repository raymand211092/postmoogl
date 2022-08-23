package bot

import (
	"context"

	"github.com/getsentry/sentry-go"
	"gitlab.com/etke.cc/postmoogle/utils"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
)

func (b *Bot) handleMailbox(ctx context.Context, evt *event.Event, command []string) {
	if len(command) == 1 {
		b.getMailbox(ctx, evt)
		return
	}
	b.setMailbox(ctx, evt, command[1])
}

func (b *Bot) getMailbox(ctx context.Context, evt *event.Event) {
	span := sentry.StartSpan(ctx, "http.server", sentry.TransactionName("getMailbox"))
	defer span.Finish()

	cfg, err := b.getSettings(span.Context(), evt.RoomID)
	if err != nil {
		b.Error(span.Context(), evt.RoomID, "failed to retrieve setting: %v", err)
		return
	}

	if cfg.Mailbox == "" {
		b.Notice(span.Context(), evt.RoomID, "mailbox name is not set")
		return
	}

	content := format.RenderMarkdown("Mailbox of this room is **"+cfg.Mailbox+"@"+b.domain+"**", true, true)
	content.MsgType = event.MsgNotice
	_, err = b.lp.Send(evt.RoomID, content)
	if err != nil {
		b.Error(span.Context(), evt.RoomID, "cannot send message: %v", err)
	}
}

func (b *Bot) setMailbox(ctx context.Context, evt *event.Event, mailbox string) {
	span := sentry.StartSpan(ctx, "http.server", sentry.TransactionName("setMailbox"))
	defer span.Finish()

	mailbox = utils.Mailbox(mailbox)
	existingID, ok := b.GetMapping(ctx, mailbox)
	if ok && existingID != "" && existingID != evt.RoomID {
		content := format.RenderMarkdown("Mailbox "+mailbox+"@"+b.domain+" already taken", true, true)
		content.MsgType = event.MsgNotice
		_, err := b.lp.Send(evt.RoomID, content)
		if err != nil {
			b.Error(span.Context(), evt.RoomID, "cannot send message: %v", err)
		}
	}
	cfg, err := b.getSettings(span.Context(), evt.RoomID)
	if err != nil {
		b.Error(span.Context(), evt.RoomID, "failed to retrieve setting: %v", err)
		return
	}

	if !cfg.Allowed(b.noowner, evt.Sender) {
		b.Notice(span.Context(), evt.RoomID, "you don't have permission to do that")
		return
	}

	cfg.Owner = evt.Sender
	cfg.Mailbox = mailbox
	err = b.setSettings(span.Context(), evt.RoomID, cfg)
	if err != nil {
		b.Error(span.Context(), evt.RoomID, "cannot update settings: %v", err)
		return
	}

	b.roomsmu.Lock()
	b.rooms[mailbox] = evt.RoomID
	b.roomsmu.Unlock()

	content := format.RenderMarkdown("Mailbox of this room set to **"+cfg.Mailbox+"@"+b.domain+"**", true, true)
	content.MsgType = event.MsgNotice
	_, err = b.lp.Send(evt.RoomID, content)
	if err != nil {
		b.Error(span.Context(), evt.RoomID, "cannot send message: %v", err)
	}
}
