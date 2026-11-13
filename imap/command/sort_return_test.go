package command

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func parseSortCommand(line string) (Command, error) {
	return NewParser(rfcparser.NewScanner(bytes.NewReader(toIMAPLine(line)))).Parse()
}

func TestParser_SortCommand(t *testing.T) {
	cmd, err := parseSortCommand(`tag SORT (REVERSE DATE subject) UTF-8 UNSEEN`)
	require.NoError(t, err)
	require.Equal(t, Command{Tag: "tag", Payload: &Sort{
		SortKeys: []SortKey{{Reverse: true, Field: "DATE"}, {Field: "SUBJECT"}},
		Charset:  "UTF-8",
		Keys:     []SearchKey{&SearchKeyUnseen{}},
	}}, cmd)
}

func TestParser_SortCommandWithReturnOptions(t *testing.T) {
	cmd, err := parseSortCommand(`tag SORT RETURN (min COUNT) (DATE) UTF-8 ALL`)
	require.NoError(t, err)
	require.Equal(t, Command{Tag: "tag", Payload: &Sort{
		SortKeys: []SortKey{{Field: "DATE"}},
		Charset:  "UTF-8",
		Keys:     []SearchKey{&SearchKeyAll{}},
		Return:   []SearchReturnOption{SearchReturnMin, SearchReturnCount},
	}}, cmd)
}

func TestParser_SortCommandWithEmptyReturnOptionsMeansAll(t *testing.T) {
	cmd, err := parseSortCommand(`tag SORT RETURN () (DATE) UTF-8 ALL`)
	require.NoError(t, err)
	require.Equal(t, []SearchReturnOption{SearchReturnAll}, cmd.Payload.(*Sort).Return)
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
