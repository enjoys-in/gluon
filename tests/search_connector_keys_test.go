package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func TestSearchConnectorIsGivenTheKeysAboutMessageContent(t *testing.T) {
	date := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	subject := &command.SearchKeySubject{Value: "a"}
	from := &command.SearchKeyFrom{Value: "b"}

	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		for i, tc := range []struct {
			search string
			key    command.SearchKey
		}{
			{search: `SUBJECT a`, key: subject},
			{search: `FROM b`, key: from},
			{search: `TO a`, key: &command.SearchKeyTo{Value: "a"}},
			{search: `CC a`, key: &command.SearchKeyCC{Value: "a"}},
			{search: `BCC a`, key: &command.SearchKeyBCC{Value: "a"}},
			{search: `HEADER X-Key a`, key: &command.SearchKeyHeader{Field: "X-Key", Value: "a"}},
			{search: `BODY a`, key: &command.SearchKeyBody{Value: "a"}},
			{search: `TEXT a`, key: &command.SearchKeyText{Value: "a"}},
			{search: `SINCE 1-Jan-2020`, key: &command.SearchKeySince{Value: date}},
			{search: `BEFORE 1-Jan-2020`, key: &command.SearchKeyBefore{Value: date}},
			{search: `ON 1-Jan-2020`, key: &command.SearchKeyOn{Value: date}},
			{search: `SENTSINCE 1-Jan-2020`, key: &command.SearchKeySentSince{Value: date}},
			{search: `SENTBEFORE 1-Jan-2020`, key: &command.SearchKeySentBefore{Value: date}},
			{search: `SENTON 1-Jan-2020`, key: &command.SearchKeySentOn{Value: date}},
			{search: `OLDER 60`, key: &command.SearchKeyOlder{Value: 60}},
			{search: `YOUNGER 60`, key: &command.SearchKeyYounger{Value: 60}},
			{search: `NOT SUBJECT a`, key: &command.SearchKeyNot{Key: subject}},
			{search: `OR SUBJECT a FROM b`, key: &command.SearchKeyOr{Key1: subject, Key2: from}},
			{search: `(SUBJECT a FROM b)`, key: &command.SearchKeyList{Keys: []command.SearchKey{subject, from}}},
		} {
			tag := fmt.Sprintf("A%03d", i)

			c.Cf(`%v SEARCH %v`, tag, tc.search)
			c.Sx(`^\* SEARCH`)
			c.OK(tag)

			searches := f.conn.searches()
			require.Len(t, searches, i+1, tc.search)
			require.Equal(t, []command.SearchKey{tc.key}, searches[i], tc.search)
		}
	})
}

func TestSearchConnectorIsNotGivenTheKeysGluonAnswers(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		for i, search := range []string{
			`ALL`,
			`UNSEEN`,
			`DELETED`,
			`KEYWORD a`,
			`RECENT`,
			`UID 1`,
			`1:2`,
			`LARGER 10`,
			`SMALLER 10`,
			`NOT UNSEEN`,
			`OR UNSEEN SEEN`,
			// A key that mixes both kinds stays with Gluon as a whole.
			`OR UNSEEN SUBJECT a`,
			`(SUBJECT a UNSEEN)`,
			`NOT OR SUBJECT a NOT UNSEEN`,
		} {
			tag := fmt.Sprintf("A%03d", i)

			c.Cf(`%v SEARCH %v`, tag, search)
			c.Sx(`^\* SEARCH`)
			c.OK(tag)

			require.Empty(t, f.conn.searches(), search)
		}
	})
}
