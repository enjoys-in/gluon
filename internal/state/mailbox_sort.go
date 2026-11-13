package state

import (
	"bytes"
	"context"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/ProtonMail/gluon/async"
	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/contexts"
	"github.com/ProtonMail/gluon/rfc5322"
	"github.com/ProtonMail/gluon/rfc822"
	"github.com/bradenaw/juniper/parallel"
	"github.com/bradenaw/juniper/xslices"
	"golang.org/x/text/encoding"
)

// sortEntry holds the pre-extracted sort keys for a single message.
type sortEntry struct {
	seq    imap.SeqID
	uid    imap.UID
	fields map[string]string // uppercase field name -> extracted value
	size   int
}

// Sort filters messages with the given search keys, then sorts them per RFC 5256.
func (m *Mailbox) Sort(ctx context.Context, sortKeys []command.SortKey, searchKeys []command.SearchKey, decoder *encoding.Decoder) ([]uint32, error) {
	var mapFn func(sortEntry) uint32

	if contexts.IsUID(ctx) {
		mapFn = func(s sortEntry) uint32 {
			return uint32(s.uid)
		}
	} else {
		mapFn = func(s sortEntry) uint32 {
			return uint32(s.seq)
		}
	}

	// First, filter messages using existing search infrastructure.
	op, err := buildSearchOpListWithKeys(m, searchKeys, decoder)
	if err != nil {
		return nil, err
	}

	msgCount := m.snap.len()

	type matchedMsg struct {
		msg   snapMsgWithSeq
		match bool
	}

	matched := make([]matchedMsg, msgCount)

	activeRequests := atomic.AddInt32(&totalActiveSearchRequests, 1)
	defer atomic.AddInt32(&totalActiveSearchRequests, -1)

	var parallelismCount int

	if contexts.IsParallelismDisabledCtx(ctx) {
		parallelismCount = 1
	} else {
		parallelismCount = runtime.NumCPU() / int(activeRequests)
	}

	if parallelismCount < 1 {
		parallelismCount = 1
	}

	if err := parallel.DoContext(ctx, parallelismCount, msgCount, func(ctx context.Context, i int) error {
		defer async.HandlePanic(m.state.panicHandler)

		msg, ok := m.snap.messages.getWithSeqID(imap.SeqID(uint32(i + 1)))
		if !ok {
			return nil
		}

		matches, err := applySearch(ctx, m, msg, op)
		if err != nil {
			return err
		}

		matched[i] = matchedMsg{msg: msg, match: matches}

		return nil
	}); err != nil {
		return nil, err
	}

	// Collect matched messages.
	var matchedMsgs []snapMsgWithSeq

	for _, m := range matched {
		if m.match {
			matchedMsgs = append(matchedMsgs, m.msg)
		}
	}

	if len(matchedMsgs) == 0 {
		return nil, nil
	}

	// Determine which fields we need to extract.
	needsHeader := false
	needsSize := false

	for _, sk := range sortKeys {
		switch sk.Field {
		case "DATE", "FROM", "TO", "CC", "SUBJECT":
			needsHeader = true
		case "SIZE":
			needsSize = true
		case "ARRIVAL":
			needsSize = true // we use the DB date/size fetch
		}
	}

	// Build sort entries with extracted fields.
	entries := make([]sortEntry, len(matchedMsgs))

	if err := parallel.DoContext(ctx, parallelismCount, len(matchedMsgs), func(ctx context.Context, i int) error {
		defer async.HandlePanic(m.state.panicHandler)

		msg := matchedMsgs[i]
		entry := sortEntry{
			seq:    msg.Seq,
			uid:    msg.UID,
			fields: make(map[string]string),
		}

		// Get date and size from DB if needed.
		if needsSize || hasField(sortKeys, "ARRIVAL") {
			if err := stateDBRead(ctx, m.state, func(ctx context.Context, client db.ReadOnly) error {
				date, size, err := client.GetMessageDateAndSize(ctx, msg.ID.InternalID)
				if err != nil {
					return err
				}

				entry.size = size
				entry.fields["ARRIVAL"] = date.Format("20060102150405")

				return nil
			}); err != nil {
				return err
			}
		}

		if needsHeader {
			literal, err := m.state.getLiteral(ctx, msg.ID)
			if err != nil {
				return err
			}

			headerBytes, _ := rfc822.Split(literal)

			header, err := rfc822.NewHeader(headerBytes)
			if err != nil {
				return err
			}

			// Extract all header-based fields.
			for _, sk := range sortKeys {
				switch sk.Field {
				case "DATE":
					dateStr := header.Get("Date")
					if dt, err := rfc5322.ParseDateTime(dateStr); err == nil {
						entry.fields["DATE"] = dt.UTC().Format("20060102150405")
					} else {
						entry.fields["DATE"] = ""
					}

				case "FROM":
					entry.fields["FROM"] = extractSortAddress(header.Get("From"))

				case "TO":
					entry.fields["TO"] = extractSortAddress(header.Get("To"))

				case "CC":
					entry.fields["CC"] = extractSortAddress(header.Get("Cc"))

				case "SUBJECT":
					entry.fields["SUBJECT"] = extractBaseSubject(header.Get("Subject"))
				}
			}

			if hasField(sortKeys, "SIZE") && !needsSize {
				entry.size = len(literal)
			}
		}

		entries[i] = entry

		return nil
	}); err != nil {
		return nil, err
	}

	// Sort the entries.
	sort.SliceStable(entries, func(i, j int) bool {
		return compareSortEntries(entries[i], entries[j], sortKeys)
	})

	return xslices.Map(entries, mapFn), nil
}

func hasField(keys []command.SortKey, field string) bool {
	for _, k := range keys {
		if k.Field == field {
			return true
		}
	}

	return false
}

func compareSortEntries(a, b sortEntry, sortKeys []command.SortKey) bool {
	for _, sk := range sortKeys {
		var cmp int

		switch sk.Field {
		case "SIZE":
			cmp = a.size - b.size
		default:
			cmp = strings.Compare(a.fields[sk.Field], b.fields[sk.Field])
		}

		if cmp == 0 {
			continue
		}

		if sk.Reverse {
			return cmp > 0
		}

		return cmp < 0
	}

	// RFC 5256: when all sort keys are equal, sort by ascending sequence number.
	return a.seq < b.seq
}

// extractSortAddress extracts the sort address per RFC 5256 §2.2:
// addr-mailbox if addr-name is empty, otherwise addr-name (lowercased).
func extractSortAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}

	// Try simple extraction: "Name <email>" or just "email".
	if idx := strings.Index(addr, "<"); idx >= 0 {
		name := strings.TrimSpace(addr[:idx])
		if name != "" {
			return strings.ToLower(name)
		}

		email := strings.TrimRight(strings.TrimLeft(addr[idx:], "<"), ">")

		return strings.ToLower(strings.TrimSpace(email))
	}

	return strings.ToLower(addr)
}

// baseSubjectRe matches Re:/Fwd:/Fw: prefixes with optional [list] markers per RFC 5256 §2.1.
var baseSubjectRe = regexp.MustCompile(`(?i)^(\s*(re|fwd|fw)\s*(\[\d+\])?\s*:\s*|\s*\[.*?\]\s*)+`)
var trailingFwd = regexp.MustCompile(`(?i)\s*\(fwd\)\s*$`)

// extractBaseSubject extracts the "base subject" per RFC 5256 §2.1.
func extractBaseSubject(subject string) string {
	// Step 1: Decode and unfold.
	subject = strings.ReplaceAll(subject, "\r\n", "")
	subject = strings.ReplaceAll(subject, "\n", "")

	// Collapse whitespace.
	subject = collapseWhitespace(subject)
	subject = strings.TrimSpace(subject)

	// Iteratively remove prefixes and suffixes.
	for {
		prev := subject

		// Remove trailing "(fwd)".
		subject = trailingFwd.ReplaceAllString(subject, "")
		subject = strings.TrimSpace(subject)

		// Remove leading prefixes: Re:, Fwd:, Fw:, [list]
		subject = baseSubjectRe.ReplaceAllString(subject, "")
		subject = strings.TrimSpace(subject)

		if subject == prev {
			break
		}
	}

	return strings.ToLower(subject)
}

func collapseWhitespace(s string) string {
	var buf bytes.Buffer

	inSpace := false

	for _, r := range s {
		if r == ' ' || r == '\t' {
			if !inSpace {
				buf.WriteByte(' ')
				inSpace = true
			}
		} else {
			buf.WriteRune(r)
			inSpace = false
		}
	}

	return buf.String()
}
