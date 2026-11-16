package response

import (
	"strconv"
	"strings"
)

type itemPreview struct {
	text string
}

// ItemPreview is the PREVIEW fetch item (RFC 8970).
func ItemPreview(text string) *itemPreview {
	return &itemPreview{
		text: text,
	}
}

func (c *itemPreview) String() string {
	// A quoted string only holds 7-bit text without line breaks. Other text is sent as a literal.
	for i := 0; i < len(c.text); i++ {
		if ch := c.text[i]; ch < 0x20 || ch > 0x7e {
			return "PREVIEW {" + strconv.Itoa(len(c.text)) + "}\r\n" + c.text
		}
	}

	return `PREVIEW "` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(c.text) + `"`
}
