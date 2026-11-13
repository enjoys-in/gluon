package state

import (
	"testing"

	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
	"github.com/stretchr/testify/require"
)

func TestMatchesAnyPattern(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		patterns []string
		mailbox  string
		want     bool
	}{
		{name: "star matches everything", patterns: []string{"*"}, mailbox: "Work/Reports", want: true},
		{name: "percent matches the top level", patterns: []string{"%"}, mailbox: "Work", want: true},
		{name: "percent does not match a child", patterns: []string{"%"}, mailbox: "Work/Reports", want: false},
		{name: "second pattern matches", patterns: []string{"INBOX", "Work/%"}, mailbox: "Work/Reports", want: true},
		{name: "no pattern matches", patterns: []string{"INBOX", "Drafts"}, mailbox: "Work", want: false},
		{name: "reference is prepended", ref: "Work/", patterns: []string{"%"}, mailbox: "Work/Reports", want: true},
		{name: "inbox is case insensitive", patterns: []string{"inbox"}, mailbox: "INBOX", want: true},
		{name: "empty pattern matches no mailbox", patterns: []string{""}, mailbox: "Work", want: false},
		{name: "no patterns", mailbox: "Work", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, matchesAnyPattern(tc.ref, tc.patterns, "/", tc.mailbox))
		})
	}
}

func newTestMailbox(name string, attributes ...string) *db.MailboxWithAttr {
	return &db.MailboxWithAttr{
		Mailbox:    db.Mailbox{Name: name},
		Attributes: imap.NewFlagSet(attributes...),
	}
}

func TestAddChildAttributes(t *testing.T) {
	mailboxes := []*db.MailboxWithAttr{
		newTestMailbox("INBOX"),
		newTestMailbox("Work"),
		newTestMailbox("Work/Reports"),
		newTestMailbox("Deleted/Child"),
		newTestMailbox("Leaf", imap.AttrNoInferiors),
	}

	matches := map[string]Match{
		"":             {Name: "", Atts: imap.NewFlagSet(imap.AttrNoSelect)},
		"INBOX":        {Name: "INBOX", Atts: imap.NewFlagSet(imap.AttrUnmarked)},
		"Work":         {Name: "Work", Atts: imap.NewFlagSet(imap.AttrUnmarked)},
		"Work/Reports": {Name: "Work/Reports", Atts: imap.NewFlagSet(imap.AttrUnmarked)},
		"Deleted":      {Name: "Deleted", Atts: imap.NewFlagSet(imap.AttrNoSelect)},
		"Leaf":         {Name: "Leaf", Atts: imap.NewFlagSet(imap.AttrNoInferiors, imap.AttrUnmarked)},
		"Root/":        {Name: "Root/", Atts: imap.NewFlagSet(imap.AttrNoSelect)},
	}

	addChildAttributes(matches, mailboxes, "/", map[string]struct{}{"Root/": {}})

	require.Equal(t, []string{imap.AttrNoSelect}, matches[""].Atts.ToSlice())
	require.Equal(t, []string{imap.AttrHasNoChildren, imap.AttrUnmarked}, matches["INBOX"].Atts.ToSlice())
	require.Equal(t, []string{imap.AttrHasChildren, imap.AttrUnmarked}, matches["Work"].Atts.ToSlice())
	require.Equal(t, []string{imap.AttrHasNoChildren, imap.AttrUnmarked}, matches["Work/Reports"].Atts.ToSlice())
	require.Equal(t, []string{imap.AttrHasChildren, imap.AttrNoSelect}, matches["Deleted"].Atts.ToSlice())
	require.Equal(t, []string{imap.AttrNoInferiors, imap.AttrUnmarked}, matches["Leaf"].Atts.ToSlice())
	require.Equal(t, []string{imap.AttrNoSelect}, matches["Root/"].Atts.ToSlice())
}

func TestAddChildAttributesWithoutDelimiter(t *testing.T) {
	mailboxes := []*db.MailboxWithAttr{newTestMailbox("Work"), newTestMailbox("Work/Reports")}

	matches := map[string]Match{
		"Work": {Name: "Work", Atts: imap.NewFlagSet(imap.AttrUnmarked)},
	}

	addChildAttributes(matches, mailboxes, "", nil)

	require.Equal(t, []string{imap.AttrHasNoChildren, imap.AttrUnmarked}, matches["Work"].Atts.ToSlice())
}
