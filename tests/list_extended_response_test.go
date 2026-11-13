package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/internal/response"

	"github.com/ProtonMail/gluon/imap"
	"github.com/stretchr/testify/assert"
)

func TestResponseListWithChildAttribute(t *testing.T) {
	assert.Equal(
		t,
		`* LIST (\HasChildren \Unmarked) "/" "Work"`,
		response.List().WithName("Work").WithDelimiter("/").WithAttributes(imap.NewFlagSet(imap.AttrHasChildren, imap.AttrUnmarked)).String(),
	)
}

func TestResponseListWithChildInfo(t *testing.T) {
	assert.Equal(
		t,
		`* LIST (\HasChildren \NonExistent) "/" "Fruit" ("CHILDINFO" ("SUBSCRIBED"))`,
		response.List().
			WithName("Fruit").
			WithDelimiter("/").
			WithAttributes(imap.NewFlagSet(imap.AttrHasChildren, imap.AttrNonExistent)).
			WithChildInfo([]string{"SUBSCRIBED"}).
			String(),
	)
}

func TestResponseListWithoutChildInfoHasNoExtendedData(t *testing.T) {
	assert.Equal(
		t,
		`* LIST (\Subscribed) "/" "INBOX"`,
		response.List().WithName("INBOX").WithDelimiter("/").WithAttributes(imap.NewFlagSet(imap.AttrSubscribed)).WithChildInfo(nil).String(),
	)
}
