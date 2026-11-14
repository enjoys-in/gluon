package state

import (
	"bytes"
	"encoding/base64"
	"html"
	"io"
	"mime/quotedprintable"
	"regexp"
	"strings"

	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/rfc822"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

const (
	// maxPreviewLength is the longest preview sent, in characters. RFC 8970 recommends 200.
	maxPreviewLength = 200

	// maxPreviewSource is how much of a decoded body part is read to build a preview.
	maxPreviewSource = 64 * 1024

	// maxPreviewDepth is how deep nested multipart messages are searched for text.
	maxPreviewDepth = 16
)

var (
	previewHiddenHTML = regexp.MustCompile(`(?is)<(script|style|head|title)\b.*?(</(script|style|head|title)>|$)|<!--.*?(-->|$)`)
	previewHTMLTag    = regexp.MustCompile(`(?s)<[^>]*(>|$)`)
)

func fetchPreview(_ snapMsgWithSeq, _ *db.Message, fc *fetchContext) (response.Item, error) {
	return response.ItemPreview(messagePreview(fc.parse())), nil
}

// messagePreview returns the first characters of the message's text: of its plain text part, or of its HTML part
// when it has no plain text. A message without text has an empty preview.
func messagePreview(root *rfc822.Section) string {
	for _, contentType := range []rfc822.MIMEType{rfc822.TextPlain, rfc822.TextHTML} {
		if text := sectionPreview(root, contentType, 0); text != "" {
			return text
		}
	}

	return ""
}

func sectionPreview(section *rfc822.Section, want rfc822.MIMEType, depth int) string {
	contentType, params, err := section.ContentType()
	if err != nil || depth > maxPreviewDepth {
		return ""
	}

	if contentType.IsMultiPart() {
		children, err := section.Children()
		if err != nil {
			return ""
		}

		for _, child := range children {
			if text := sectionPreview(child, want, depth+1); text != "" {
				return text
			}
		}

		return ""
	}

	header, err := section.ParseHeader()
	if err != nil || contentType != want {
		return ""
	}

	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(header.Get("Content-Disposition"))), "attachment") {
		return ""
	}

	text := decodePreviewSource(section.Body(), header.Get("Content-Transfer-Encoding"), params["charset"])

	if want == rfc822.TextHTML {
		text = html.UnescapeString(previewHTMLTag.ReplaceAllString(previewHiddenHTML.ReplaceAllString(text, " "), " "))
	}

	return truncatePreview(strings.Join(strings.Fields(text), " "))
}

// decodePreviewSource decodes the start of a body part into UTF-8 text. A part that is cut short or broken still
// gives the text that could be read.
func decodePreviewSource(body []byte, transferEncoding, charset string) string {
	var reader io.Reader = bytes.NewReader(body)

	switch strings.ToLower(strings.TrimSpace(transferEncoding)) {
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, reader)

	case "quoted-printable":
		reader = quotedprintable.NewReader(reader)
	}

	if charset != "" && !strings.EqualFold(charset, "utf-8") && !strings.EqualFold(charset, "us-ascii") {
		if enc, err := ianaindex.MIME.Encoding(charset); err == nil && enc != nil {
			reader = transform.NewReader(reader, enc.NewDecoder())
		}
	}

	source := make([]byte, maxPreviewSource)

	n, _ := io.ReadFull(reader, source)

	return strings.ToValidUTF8(string(source[:n]), "")
}

func truncatePreview(text string) string {
	count := 0

	for i := range text {
		if count == maxPreviewLength {
			return text[:i]
		}

		count++
	}

	return text
}
