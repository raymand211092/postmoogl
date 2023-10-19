package linkpearl

import (
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

// Send a message to the roomID and automatically try to encrypt it, if the destination room is encrypted
//
//nolint:unparam // it's public interface
func (l *Linkpearl) Send(roomID id.RoomID, content interface{}) (id.EventID, error) {
	l.log.Debug().Str("roomID", roomID.String()).Any("content", content).Msg("sending event")
	resp, err := l.api.SendMessageEvent(roomID, event.EventMessage, content)
	if err != nil {
		return "", UnwrapError(err)
	}
	return resp.EventID, nil
}

// SendNotice to a room with optional relations, markdown supported
func (l *Linkpearl) SendNotice(roomID id.RoomID, message string, relates ...*event.RelatesTo) {
	var withRelatesTo bool
	content := format.RenderMarkdown(message, true, true)
	content.MsgType = event.MsgNotice
	if len(relates) > 0 {
		withRelatesTo = true
		content.RelatesTo = relates[0]
	}

	_, err := l.Send(roomID, &content)
	if err != nil {
		l.log.Error().Err(UnwrapError(err)).Str("roomID", roomID.String()).Str("retries", "1/2").Msg("cannot send a notice into the room")
		if withRelatesTo {
			content.RelatesTo = nil
			_, err = l.Send(roomID, &content)
			if err != nil {
				l.log.Error().Err(UnwrapError(err)).Str("roomID", roomID.String()).Str("retries", "2/2").Msg("cannot send a notice into the room even without relations")
			}
		}
	}
}

// SendFile to a matrix room
func (l *Linkpearl) SendFile(roomID id.RoomID, req *mautrix.ReqUploadMedia, msgtype event.MessageType, relates ...*event.RelatesTo) error {
	var relation *event.RelatesTo
	if len(relates) > 0 {
		relation = relates[0]
	}

	resp, err := l.GetClient().UploadMedia(*req)
	if err != nil {
		err = UnwrapError(err)
		l.log.Error().Err(err).Str("file", req.FileName).Msg("cannot upload file")
		return err
	}
	content := &event.MessageEventContent{
		MsgType:   msgtype,
		Body:      req.FileName,
		URL:       resp.ContentURI.CUString(),
		RelatesTo: relation,
	}

	_, err = l.Send(roomID, content)
	err = UnwrapError(err)
	if err != nil {
		l.log.Error().Err(err).Str("roomID", roomID.String()).Str("retries", "1/2").Msg("cannot send file into the room")
		if relation != nil {
			content.RelatesTo = nil
			_, err = l.Send(roomID, &content)
			err = UnwrapError(err)
			if err != nil {
				l.log.Error().Err(UnwrapError(err)).Str("roomID", roomID.String()).Str("retries", "2/2").Msg("cannot send file into the room even without relations")
			}
		}
	}

	return err
}
