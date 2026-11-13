package state

import (
	"testing"
	"time"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func TestGetSearchKeyOwner(t *testing.T) {
	subject := &command.SearchKeySubject{Value: "invoice"}
	unseen := &command.SearchKeyUnseen{}

	tests := []struct {
		name string
		key  command.SearchKey
		want searchKeyOwner
	}{
		{name: "subject", key: subject, want: searchKeyOwnerConnector},
		{name: "from", key: &command.SearchKeyFrom{Value: "a"}, want: searchKeyOwnerConnector},
		{name: "to", key: &command.SearchKeyTo{Value: "a"}, want: searchKeyOwnerConnector},
		{name: "cc", key: &command.SearchKeyCC{Value: "a"}, want: searchKeyOwnerConnector},
		{name: "bcc", key: &command.SearchKeyBCC{Value: "a"}, want: searchKeyOwnerConnector},
		{name: "header", key: &command.SearchKeyHeader{Field: "X", Value: "a"}, want: searchKeyOwnerConnector},
		{name: "body", key: &command.SearchKeyBody{Value: "a"}, want: searchKeyOwnerConnector},
		{name: "text", key: &command.SearchKeyText{Value: "a"}, want: searchKeyOwnerConnector},
		{name: "since", key: &command.SearchKeySince{Value: time.Now()}, want: searchKeyOwnerConnector},
		{name: "sent before", key: &command.SearchKeySentBefore{Value: time.Now()}, want: searchKeyOwnerConnector},

		{name: "all", key: &command.SearchKeyAll{}, want: searchKeyOwnerGluon},
		{name: "unseen", key: unseen, want: searchKeyOwnerGluon},
		{name: "deleted", key: &command.SearchKeyDeleted{}, want: searchKeyOwnerGluon},
		{name: "keyword", key: &command.SearchKeyKeyword{Value: "a"}, want: searchKeyOwnerGluon},
		{name: "recent", key: &command.SearchKeyRecent{}, want: searchKeyOwnerGluon},
		{name: "uid set", key: &command.SearchKeyUID{}, want: searchKeyOwnerGluon},
		{name: "sequence set", key: &command.SearchKeySeqSet{}, want: searchKeyOwnerGluon},
		{name: "larger", key: &command.SearchKeyLarger{Value: 10}, want: searchKeyOwnerGluon},
		{name: "smaller", key: &command.SearchKeySmaller{Value: 10}, want: searchKeyOwnerGluon},

		{name: "not content", key: &command.SearchKeyNot{Key: subject}, want: searchKeyOwnerConnector},
		{name: "not flag", key: &command.SearchKeyNot{Key: unseen}, want: searchKeyOwnerGluon},
		{name: "or of content", key: &command.SearchKeyOr{Key1: subject, Key2: subject}, want: searchKeyOwnerConnector},
		{name: "or of flags", key: &command.SearchKeyOr{Key1: unseen, Key2: unseen}, want: searchKeyOwnerGluon},
		{name: "or of both", key: &command.SearchKeyOr{Key1: unseen, Key2: subject}, want: searchKeyOwnerMixed},
		{name: "list of content", key: &command.SearchKeyList{Keys: []command.SearchKey{subject, subject}}, want: searchKeyOwnerConnector},
		{name: "list of both", key: &command.SearchKeyList{Keys: []command.SearchKey{subject, unseen}}, want: searchKeyOwnerMixed},
		{name: "empty list", key: &command.SearchKeyList{}, want: searchKeyOwnerGluon},
		{
			name: "nested mix",
			key:  &command.SearchKeyNot{Key: &command.SearchKeyOr{Key1: subject, Key2: &command.SearchKeyNot{Key: unseen}}},
			want: searchKeyOwnerMixed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, getSearchKeyOwner(tc.key))
		})
	}
}
