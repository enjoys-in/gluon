package command

import (
	"fmt"

	"github.com/ProtonMail/gluon/rfcparser"
)

// UIDBatches asks for the UID ranges that split the selected mailbox into batches of BatchSize messages, newest
// first. From and To select batches by their number, counted from one; both are zero when all batches are wanted.
type UIDBatches struct {
	BatchSize int
	From, To  int
}

func (l UIDBatches) HasRange() bool {
	return l.From != 0
}

func (l UIDBatches) String() string {
	if l.HasRange() {
		return fmt.Sprintf("UIDBATCHES %v %v:%v", l.BatchSize, l.From, l.To)
	}

	return fmt.Sprintf("UIDBATCHES %v", l.BatchSize)
}

func (l UIDBatches) SanitizedString() string {
	return l.String()
}

type UIDBatchesCommandParser struct{}

func (UIDBatchesCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// uidbatches = "UIDBATCHES" SP nz-number [SP nz-number ":" nz-number]
	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after command"); err != nil {
		return nil, err
	}

	batchSize, err := ParseNZNumber(p)
	if err != nil {
		return nil, err
	}

	uidBatches := &UIDBatches{BatchSize: batchSize}

	if ok, err := p.Matches(rfcparser.TokenTypeSP); err != nil {
		return nil, err
	} else if !ok {
		return uidBatches, nil
	}

	if uidBatches.From, err = ParseNZNumber(p); err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeColon, "expected : between the batch numbers"); err != nil {
		return nil, err
	}

	if uidBatches.To, err = ParseNZNumber(p); err != nil {
		return nil, err
	}

	if uidBatches.From > uidBatches.To {
		return nil, p.MakeError("batch ranges must be ordered from lowest to highest")
	}

	return uidBatches, nil
}
