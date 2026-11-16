package state

import (
	"time"

	"github.com/ProtonMail/gluon/imap/command"
)

func buildSearchOpOlder(key *command.SearchKeyOlder) (*buildSearchOpResult, error) {
	limit := time.Now().Add(-time.Duration(key.Value) * time.Second)

	op := func(s *searchData) (bool, error) {
		return s.dbMessage.date.Before(limit), nil
	}

	return newBuildSearchOpResult(op, needsDBMessage()), nil
}

func buildSearchOpYounger(key *command.SearchKeyYounger) (*buildSearchOpResult, error) {
	limit := time.Now().Add(-time.Duration(key.Value) * time.Second)

	op := func(s *searchData) (bool, error) {
		return !s.dbMessage.date.Before(limit), nil
	}

	return newBuildSearchOpResult(op, needsDBMessage()), nil
}
