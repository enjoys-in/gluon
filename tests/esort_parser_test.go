package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap/command"

	"github.com/stretchr/testify/require"
)

func parseSortCommand(line string) (command.Command, error) {
	return parseCommand(line)
}

func TestParser_SortCommand(t *testing.T) {
	cmd, err := parseSortCommand(`tag SORT (REVERSE DATE subject) UTF-8 UNSEEN`)
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.Sort{
		SortKeys: []command.SortKey{{Reverse: true, Field: "DATE"}, {Field: "SUBJECT"}},
		Charset:  "UTF-8",
		Keys:     []command.SearchKey{&command.SearchKeyUnseen{}},
	}}, cmd)
}

func TestParser_SortCommandWithReturnOptions(t *testing.T) {
	cmd, err := parseSortCommand(`tag SORT RETURN (min COUNT) (DATE) UTF-8 ALL`)
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.Sort{
		SortKeys: []command.SortKey{{Field: "DATE"}},
		Charset:  "UTF-8",
		Keys:     []command.SearchKey{&command.SearchKeyAll{}},
		Return:   []command.SearchReturnOption{command.SearchReturnMin, command.SearchReturnCount},
	}}, cmd)
}

func TestParser_SortCommandWithEmptyReturnOptionsMeansAll(t *testing.T) {
	cmd, err := parseSortCommand(`tag SORT RETURN () (DATE) UTF-8 ALL`)
	require.NoError(t, err)
	require.Equal(t, []command.SearchReturnOption{command.SearchReturnAll}, cmd.Payload.(*command.Sort).Return)
}

func TestParser_SortCommandWithReturnOptionsFailures(t *testing.T) {
	for _, line := range []string{
		`tag SORT RETURN (SAVE) (DATE) UTF-8 ALL`,
		`tag SORT RETURN (DATE) UTF-8 ALL`,
		`tag SORT RETURN (ALL)`,
		`tag SORT RETURN (ALL)(DATE) UTF-8 ALL`,
		`tag SORT RETURN ALL (DATE) UTF-8 ALL`,
		`tag SORT RETURNS (ALL) (DATE) UTF-8 ALL`,
		`tag SORT RETURN (ALL) (DATE) UTF-8`,
		`tag SORT DATE UTF-8 ALL`,
	} {
		_, err := parseSortCommand(line)
		require.Error(t, err, line)
	}
}
