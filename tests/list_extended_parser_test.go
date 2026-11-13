package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap/command"

	"github.com/stretchr/testify/require"
)

func parseList(t *testing.T, line string) *command.List {
	t.Helper()

	cmd, err := parseCommand(line)
	require.NoError(t, err)

	list, ok := cmd.Payload.(*command.List)
	require.True(t, ok)

	return list
}

func TestParser_ListBasicIsNotExtended(t *testing.T) {
	list := parseList(t, `tag LIST "" "*"`)

	require.False(t, list.Extended.Used)
	require.Equal(t, []string{"*"}, list.Patterns())
}

func TestParser_ListExtendedSelectOptions(t *testing.T) {
	list := parseList(t, `tag LIST (SUBSCRIBED REMOTE RECURSIVEMATCH SPECIAL-USE) "" "*"`)

	require.True(t, list.Extended.Used)
	require.Equal(t, command.ListSelectOptions{Subscribed: true, Remote: true, RecursiveMatch: true, SpecialUse: true}, list.Extended.Select)
	require.Equal(t, command.ListReturnOptions{}, list.Extended.Return)
	require.Empty(t, list.Mailbox)
	require.Equal(t, "*", list.ListMailbox)
}

func TestParser_ListExtendedEmptySelectOptions(t *testing.T) {
	list := parseList(t, `tag LIST () "ref" %`)

	require.True(t, list.Extended.Used)
	require.Equal(t, command.ListSelectOptions{}, list.Extended.Select)
	require.Equal(t, "ref", list.Mailbox)
	require.Equal(t, "%", list.ListMailbox)
}

func TestParser_ListExtendedOptionsAreCaseInsensitive(t *testing.T) {
	list := parseList(t, `tag list (subscribed special-use) "" "*" return (children Subscribed)`)

	require.Equal(t, command.ListSelectOptions{Subscribed: true, SpecialUse: true}, list.Extended.Select)
	require.Equal(t, command.ListReturnOptions{Subscribed: true, Children: true}, list.Extended.Return)
}

func TestParser_ListExtendedMultiplePatterns(t *testing.T) {
	list := parseList(t, `tag LIST "" ("INBOX" Drafts "Sent/%")`)

	require.True(t, list.Extended.Used)
	require.Equal(t, "INBOX", list.ListMailbox)
	require.Equal(t, []string{"INBOX", "Drafts", "Sent/%"}, list.Patterns())
}

func TestParser_ListExtendedSinglePatternInList(t *testing.T) {
	list := parseList(t, `tag LIST "" (*)`)

	require.True(t, list.Extended.Used)
	require.Equal(t, []string{"*"}, list.Patterns())
}

func TestParser_ListExtendedReturnOptions(t *testing.T) {
	list := parseList(t, `tag LIST "" "*" RETURN (SUBSCRIBED CHILDREN SPECIAL-USE)`)

	require.True(t, list.Extended.Used)
	require.Equal(t, command.ListReturnOptions{Subscribed: true, Children: true, SpecialUse: true}, list.Extended.Return)
}

func TestParser_ListExtendedEmptyReturnOptions(t *testing.T) {
	list := parseList(t, `tag LIST "" "*" RETURN ()`)

	require.True(t, list.Extended.Used)
	require.Equal(t, command.ListReturnOptions{}, list.Extended.Return)
}

func TestParser_ListExtendedReturnStatus(t *testing.T) {
	list := parseList(t, `tag LIST "" % RETURN (CHILDREN STATUS (MESSAGES UNSEEN) SPECIAL-USE)`)

	require.Equal(t, command.ListReturnOptions{
		Children:   true,
		SpecialUse: true,
		Status:     []command.StatusAttribute{command.StatusAttributeMessages, command.StatusAttributeUnseen},
	}, list.Extended.Return)
}

func TestParser_ListExtendedEverything(t *testing.T) {
	list := parseList(t, `tag LIST (SUBSCRIBED RECURSIVEMATCH) "" ("INBOX" "Work/*") RETURN (STATUS (UIDNEXT))`)

	require.Equal(t, command.ListSelectOptions{Subscribed: true, RecursiveMatch: true}, list.Extended.Select)
	require.Equal(t, []string{"INBOX", "Work/*"}, list.Patterns())
	require.Equal(t, []command.StatusAttribute{command.StatusAttributeUIDNext}, list.Extended.Return.Status)
}

func TestParser_ListExtendedInvalid(t *testing.T) {
	for name, line := range map[string]string{
		"unknown selection option":            `tag LIST (BOGUS) "" "*"`,
		"recursivematch alone":                `tag LIST (RECURSIVEMATCH) "" "*"`,
		"recursivematch with remote only":     `tag LIST (REMOTE RECURSIVEMATCH) "" "*"`,
		"recursivematch with special-use":     `tag LIST (SPECIAL-USE RECURSIVEMATCH) "" "*"`,
		"unterminated selection options":      `tag LIST (SUBSCRIBED "" "*"`,
		"selection options without reference": `tag LIST (SUBSCRIBED)`,
		"empty pattern list":                  `tag LIST "" ()`,
		"unterminated pattern list":           `tag LIST "" ("INBOX"`,
		"unknown return option":               `tag LIST "" "*" RETURN (BOGUS)`,
		"trailing word is not RETURN":         `tag LIST "" "*" BOGUS (CHILDREN)`,
		"return without options":              `tag LIST "" "*" RETURN`,
		"return options without parentheses":  `tag LIST "" "*" RETURN CHILDREN`,
		"unterminated return options":         `tag LIST "" "*" RETURN (CHILDREN`,
		"status without attributes":           `tag LIST "" "*" RETURN (STATUS)`,
		"status with empty attributes":        `tag LIST "" "*" RETURN (STATUS ())`,
		"status with unknown attribute":       `tag LIST "" "*" RETURN (STATUS (BOGUS))`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseCommand(line)
			require.Error(t, err)
		})
	}
}
