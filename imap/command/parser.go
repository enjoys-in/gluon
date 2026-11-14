package command

import (
	"fmt"
	"strings"

	"github.com/ProtonMail/gluon/rfcparser"
)

type Builder interface {
	FromParser(p *rfcparser.Parser) (Payload, error)
}

type parserBuilder struct {
	continuationCallback    func(string) error
	disableIMAPAuthenticate bool
}

type Option interface {
	config(*parserBuilder)
}

type withLiteralContinuationCallback struct {
	callback func(string) error
}

func (opt withLiteralContinuationCallback) config(builder *parserBuilder) {
	builder.continuationCallback = opt.callback
}

func WithLiteralContinuationCallback(callback func(string) error) Option {
	return &withLiteralContinuationCallback{
		callback: callback,
	}
}

type withDisableIMAPAuthenticate struct{}

func (opt withDisableIMAPAuthenticate) config(builder *parserBuilder) {
	builder.disableIMAPAuthenticate = true
}

func WithDisableIMAPAuthenticate() Option {
	return &withDisableIMAPAuthenticate{}
}

// Parser parses IMAP Commands.
type Parser struct {
	parser   *rfcparser.Parser
	scanner  *rfcparser.Scanner
	commands map[string]Builder
	lastTag  string
	lastCmd  string
}

func NewParser(s *rfcparser.Scanner, options ...Option) *Parser {
	var builder parserBuilder
	for _, option := range options {
		option.config(&builder)
	}

	commands := map[string]Builder{
		"list":         &ListCommandParser{},
		"append":       &AppendCommandParser{},
		"search":       &SearchCommandParser{},
		"fetch":        &FetchCommandParser{},
		"capability":   &CapabilityCommandParser{},
		"idle":         &IdleCommandParser{},
		"noop":         &NoopCommandParser{},
		"logout":       &LogoutCommandParser{},
		"check":        &CheckCommandParser{},
		"close":        &CloseCommandParser{},
		"expunge":      &ExpungeCommandParser{},
		"unselect":     &UnselectCommandParser{},
		"starttls":     &StartTLSCommandParser{},
		"status":       &StatusCommandParser{},
		"select":       &SelectCommandParser{},
		"examine":      &ExamineCommandParser{},
		"create":       &CreateCommandParser{},
		"delete":       &DeleteCommandParser{},
		"subscribe":    &SubscribeCommandParser{},
		"unsubscribe":  &UnsubscribeCommandParser{},
		"rename":       &RenameCommandParser{},
		"lsub":         &LSubCommandParser{},
		"login":        &LoginCommandParser{},
		"store":        &StoreCommandParser{},
		"copy":         &CopyCommandParser{},
		"move":         &MoveCommandParser{},
		"uid":          NewUIDCommandParser(),
		"id":           &IDCommandParser{},
		"getquota":     &GetQuotaCommandParser{},
		"getquotaroot": &GetQuotaRootCommandParser{},
		"sort":         &SortCommandParser{},
		"thread":       &ThreadCommandParser{},
		"namespace":    &NamespaceCommandParser{},

		"unauthenticate": &UnauthenticateCommandParser{},
		"uidbatches":     &UIDBatchesCommandParser{},
		"setquota":       &SetQuotaCommandParser{},
	}

	if !builder.disableIMAPAuthenticate {
		commands["authenticate"] = &AuthenticateCommandParser{}
	}

	return &Parser{
		scanner:  s,
		parser:   rfcparser.NewParserWithLiteralContinuationCb(s, builder.continuationCallback),
		commands: commands,
	}
}

func (p *Parser) LastParsedTag() string {
	return p.lastTag
}

func (p *Parser) LastParsedCommand() string {
	return p.lastCmd
}

// ConsumeInvalidInput will consume all remaining scanner input until a new line has been reached.
func (p *Parser) ConsumeInvalidInput() error {
	rest, err := p.scanner.ConsumeUntilNewLine()
	if err != nil {
		return err
	}

	// The scanner reads one byte ahead, so the current token is the byte before the rest of the line.
	lookahead := p.parser.CurrentToken().Value

	// The data of a non-synchronizing literal (RFC 7888) is already on the wire and has to be discarded too.
	for {
		size, ok := nonSynchronizingLiteralSize(lookahead, rest)
		if !ok {
			return nil
		}

		if size >= rfcparser.MaxLiteralSize {
			return fmt.Errorf("literal size exceeds maximum size of 70MB")
		}

		if err := p.scanner.Discard(size); err != nil {
			return err
		}

		if rest, err = p.scanner.ConsumeUntilNewLine(); err != nil {
			return err
		}

		lookahead = 0
	}
}

// nonSynchronizingLiteralSize reads the size of a "{n+}" marker that ends the line.
func nonSynchronizingLiteralSize(lookahead byte, rest []byte) (int, bool) {
	i := len(rest) - 1

	for _, want := range [...]byte{'\n', '\r', '}', '+'} {
		if i < 0 || rest[i] != want {
			return 0, false
		}

		i--
	}

	size, digits := 0, 0

	for scale := 1; i >= 0 && rest[i] >= '0' && rest[i] <= '9'; i, scale = i-1, scale*10 {
		if digits++; digits > 9 {
			return rfcparser.MaxLiteralSize, true
		}

		size += int(rest[i]-'0') * scale
	}

	opening := lookahead
	if i >= 0 {
		opening = rest[i]
	}

	if digits == 0 || opening != '{' {
		return 0, false
	}

	return size, true
}

func (p *Parser) Parse() (Command, error) {
	result := Command{}

	p.lastTag = ""
	p.lastCmd = ""
	p.parser.ResetOffsetCounter()

	if err := p.parser.Advance(); err != nil {
		return result, err
	}

	tag, err := p.parseTag()
	if err != nil {
		return result, err
	}

	// Done command does not have a tag.
	if strings.ToLower(tag.Value) == "done" {
		p.lastCmd = "done"
		result.Tag = ""
		result.Payload = &Done{}
	} else {
		result.Tag = tag.Value
		p.lastTag = tag.Value

		if err := p.parser.Consume(rfcparser.TokenTypeSP, "Expected space after tag"); err != nil {
			return result, err
		}

		payload, err := p.parseCommand()
		if err != nil {
			return result, err
		}

		result.Payload = payload
	}

	if err := p.parser.Consume(rfcparser.TokenTypeCR, "expected CR"); err != nil {
		return Command{Tag: result.Tag}, err
	}

	// Can't fully consume the last new line here or we will hang forever as the clients don't send the next token.
	// In the next loop, the call to advance will ensure the next token in the stream gets loaded properly.
	if !p.parser.Check(rfcparser.TokenTypeLF) {
		return Command{Tag: result.Tag}, p.parser.MakeError("expected LF after CR")
	}

	return result, nil
}

func (p *Parser) parseCommand() (Payload, error) {
	var commandBytes []byte

	commandOffset := p.parser.CurrentToken().Offset

	for {
		if ok, err := p.parser.Matches(rfcparser.TokenTypeChar); err != nil {
			return nil, err
		} else if ok {
			commandBytes = append(commandBytes, rfcparser.ByteToLower(p.parser.PreviousToken().Value))
		} else {
			break
		}
	}

	p.lastCmd = string(commandBytes)

	builder, ok := p.commands[p.lastCmd]
	if !ok {
		return nil, p.parser.MakeErrorAtOffset(fmt.Sprintf("unknown command '%v'", p.lastCmd), commandOffset)
	}

	return builder.FromParser(p.parser)
}

func (p *Parser) parseTag() (rfcparser.String, error) {
	// tag             = 1*<any ASTRING-CHAR except "+">
	isTagChar := func(tt rfcparser.TokenType) bool {
		return rfcparser.IsAStringChar(tt) && tt != rfcparser.TokenTypePlus
	}

	if err := p.parser.ConsumeWith(isTagChar, "Invalid tag char detected"); err != nil {
		return rfcparser.String{}, err
	}

	tag, err := p.parser.CollectBytesWhileMatchesWithPrevWith(isTagChar)
	if err != nil {
		return rfcparser.String{}, err
	}

	return tag.IntoString(), err
}
