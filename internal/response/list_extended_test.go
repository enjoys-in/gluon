package response

import (
	"testing"

	"github.com/ProtonMail/gluon/imap"
	"github.com/stretchr/testify/assert"
)

func TestListWithChildAttribute(t *testing.T) {
	assert.Equal(
		t,
		`* LIST (\HasChildren \Unmarked) "/" "Work"`,
		List().WithName("Work").WithDelimiter("/").WithAttributes(imap.NewFlagSet(imap.AttrHasChildren, imap.AttrUnmarked)).String(),
	)
}

func TestListWithChildInfo(t *testing.T) {
	assert.Equal(
		t,
		`* LIST (\HasChildren \NonExistent) "/" "Fruit" ("CHILDINFO" ("SUBSCRIBED"))`,
		List().
			WithName("Fruit").
			WithDelimiter("/").
			WithAttributes(imap.NewFlagSet(imap.AttrHasChildren, imap.AttrNonExistent)).
			WithChildInfo([]string{"SUBSCRIBED"}).
			String(),
	)
}

func TestListWithoutChildInfoHasNoExtendedData(t *testing.T) {
	assert.Equal(
		t,
		`* LIST (\Subscribed) "/" "INBOX"`,
		List().WithName("INBOX").WithDelimiter("/").WithAttributes(imap.NewFlagSet(imap.AttrSubscribed)).WithChildInfo(nil).String(),
	)
}
