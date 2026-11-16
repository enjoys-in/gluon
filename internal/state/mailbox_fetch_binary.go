package state

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"mime/quotedprintable"
	"strings"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
)

// ErrUnknownCTE is returned for a body part whose content transfer encoding cannot be removed.
var ErrUnknownCTE = errors.New("unknown content transfer encoding")

func fetchBinary(attribute *command.FetchAttributeBinary, fc *fetchContext) (response.Item, error) {
	content, err := fetchBinaryContent(attribute.Part, fc)
	if err != nil {
		return nil, err
	}

	item := response.ItemBinary(attribute.Part, content)

	if attribute.Partial != nil {
		item.WithPartial(int(attribute.Partial.Offset), int(attribute.Partial.Count))
	}

	return item, nil
}

func fetchBinarySize(attribute *command.FetchAttributeBinarySize, fc *fetchContext) (response.Item, error) {
	content, err := fetchBinaryContent(attribute.Part, fc)
	if err != nil {
		return nil, err
	}

	return response.ItemBinarySize(attribute.Part, len(content)), nil
}

// fetchBinaryContent returns a body part without its content transfer encoding, or the whole message when no part
// is given.
func fetchBinaryContent(part []int, fc *fetchContext) ([]byte, error) {
	section := fc.parse()

	if len(part) > 0 {
		var err error

		if section, err = section.Part(part...); err != nil {
			return nil, err
		}
	}

	header, err := section.ParseHeader()
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(header.Get("Content-Transfer-Encoding"))) {
	case "", "7bit", "8bit", "binary":
		if len(part) == 0 {
			return fc.literal, nil
		}

		return section.Body(), nil

	case "base64":
		// A whole message cannot be sent without its encoding: its header would no longer describe it.
		if len(part) == 0 {
			return nil, ErrUnknownCTE
		}

		return io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewReader(section.Body())))

	case "quoted-printable":
		if len(part) == 0 {
			return nil, ErrUnknownCTE
		}

		return io.ReadAll(quotedprintable.NewReader(bytes.NewReader(section.Body())))

	default:
		return nil, ErrUnknownCTE
	}
}
