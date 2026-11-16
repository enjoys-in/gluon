package state

import (
	"context"
	"strings"

	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
)

func (state *State) actionCreateSpecialUseMailbox(
	ctx context.Context,
	tx db.Transaction,
	name string,
	attributes imap.FlagSet,
	uidValidity imap.UID,
) ([]Update, error) {
	updates, res, err := state.user.GetRemote().CreateSpecialUseMailbox(ctx, tx, strings.Split(name, state.delimiter), attributes)
	if err != nil {
		return nil, err
	}

	return updates, tx.CreateMailboxIfNotExists(ctx, res, state.delimiter, uidValidity)
}
