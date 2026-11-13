package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_CreateCommandWithSpecialUse(t *testing.T) {
	for line, expected := range map[string]*Create{
		`tag CREATE MySent (USE (\Sent))`:      {Mailbox: "MySent", SpecialUse: []string{`\Sent`}},
		`tag CREATE Mix (use (\Sent \Drafts))`: {Mailbox: "Mix", SpecialUse: []string{`\Sent`, `\Drafts`}},
		`tag CREATE "My Junk" (USE (\junk))`:   {Mailbox: "My Junk", SpecialUse: []string{`\junk`}},
		`tag CREATE Plain (USE ())`:            {Mailbox: "Plain", SpecialUse: []string{}},
		`tag CREATE Other (USE (\Important))`:  {Mailbox: "Other", SpecialUse: []string{`\Important`}},
		`tag CREATE Plain`:                     {Mailbox: "Plain"},
	} {
		cmd, err := testParseCommand(line)
		require.NoError(t, err, line)
		require.Equal(t, Command{Tag: "tag", Payload: expected}, cmd, line)
	}
}

func TestParser_CreateCommandWithSpecialUseFailures(t *testing.T) {
	for _, line := range []string{
		`tag CREATE Box (USE)`,
		`tag CREATE Box (USE (Sent))`,
		`tag CREATE Box (USE \Sent)`,
		`tag CREATE Box (USE (\Sent)`,
		`tag CREATE Box (USE (\Sent,\Drafts))`,
		`tag CREATE Box (USE (\Sent)) extra`,
		`tag CREATE Box (BOGUS (\Sent))`,
		`tag CREATE Box USE (\Sent)`,
		`tag CREATE Box ()`,
		`tag CREATE Box `,
	} {
		_, err := testParseCommand(line)
		require.Error(t, err, line)
	}
}
