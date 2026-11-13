package command

import (
	"time"

	"github.com/ProtonMail/gluon/rfcparser"
)

// AppendMessage is a message of an APPEND command that carries several of them (RFC 3502).
type AppendMessage struct {
	Flags    []string
	DateTime time.Time
	Literal  []byte
}

// Messages returns every message of the command, in the order they were sent.
func (l Append) Messages() []AppendMessage {
	return append([]AppendMessage{{Flags: l.Flags, DateTime: l.DateTime, Literal: l.Literal}}, l.More...)
}

// parseAppendMessages reads the messages that follow the first one. Together the messages may not be larger than
// the largest literal a command can carry.
func parseAppendMessages(p *rfcparser.Parser, size int) ([]AppendMessage, error) {
	// append         = "APPEND" SP mailbox 1*append-message
	// append-message = [SP flag-list] [SP date-time] SP literal
	var messages []AppendMessage

	for {
		if ok, err := p.Matches(rfcparser.TokenTypeSP); err != nil {
			return nil, err
		} else if !ok {
			return messages, nil
		}

		var message AppendMessage

		flagList, hasFlagList, err := TryParseFlagList(p)
		if err != nil {
			return nil, err
		} else if hasFlagList {
			message.Flags = flagList

			if err := p.Consume(rfcparser.TokenTypeSP, "expected space after flag list"); err != nil {
				return nil, err
			}
		}

		if !p.Check(rfcparser.TokenTypeLCurly) {
			if message.DateTime, err = ParseDateTime(p); err != nil {
				return nil, err
			}

			if err := p.Consume(rfcparser.TokenTypeSP, "expected space after date time"); err != nil {
				return nil, err
			}
		}

		if message.Literal, err = p.ParseLiteral(); err != nil {
			return nil, err
		}

		if size += len(message.Literal); size > rfcparser.MaxLiteralSize {
			return nil, p.MakeError("the messages of one APPEND exceed the maximum size of 70MB")
		}

		messages = append(messages, message)
	}
}
