package state

import (
	"context"
	"iter"
	"slices"
	"strings"

	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
	"github.com/bradenaw/juniper/xslices"
)

const ChildInfoSubscribed = "SUBSCRIBED"

// Flag sets are keyed by the lower case flag, so these skip the conversion on every lookup.
var (
	specialUseAttributes = xslices.Map(imap.SpecialUseAttributes, strings.ToLower)
	noInferiorsAttribute = strings.ToLower(imap.AttrNoInferiors)
)

type ListOptions struct {
	SelectSubscribed     bool
	SelectRecursiveMatch bool
	SelectSpecialUse     bool
	ReturnSubscribed     bool
}

type nameMatcher func(mailboxName string) (string, bool)

func (state *State) ListExtended(
	ctx context.Context,
	ref string,
	patterns []string,
	opts ListOptions,
	fn func([]Match) error,
) error {
	return stateDBRead(ctx, state, func(ctx context.Context, client db.ReadOnly) error {
		mailboxes, err := state.visibleMailboxes(ctx, client)
		if err != nil {
			return err
		}

		var (
			matches map[string]Match
			roots   = make(map[string]struct{})
		)

		if opts.SelectSubscribed {
			matches, err = state.listSubscribed(ctx, client, mailboxes, ref, patterns, opts)
		} else {
			matches, err = state.listExisting(ctx, client, mailboxes, ref, patterns, opts, roots)
		}

		if err != nil {
			return err
		}

		addChildAttributes(matches, mailboxes, state.delimiter, roots)

		result := make([]Match, 0, len(matches))
		for _, match := range matches {
			result = append(result, match)
		}

		slices.SortFunc(result, func(a, b Match) int {
			return strings.Compare(a.Name, b.Name)
		})

		return fn(result)
	})
}

func (state *State) listExisting(
	ctx context.Context,
	client db.ReadOnly,
	mailboxes []*db.MailboxWithAttr,
	ref string,
	patterns []string,
	opts ListOptions,
	roots map[string]struct{},
) (map[string]Match, error) {
	matchMailboxes := make([]matchMailbox, 0, len(mailboxes))
	for _, mbox := range mailboxes {
		matchMailboxes = append(matchMailboxes, matchMailbox{Name: mbox.Name, EntMBox: mbox})
	}

	var matches map[string]Match

	for _, pattern := range patterns {
		patternMatches, err := getMatches(ctx, client, matchMailboxes, ref, pattern, state.delimiter, false)
		if err != nil {
			return nil, err
		}

		// An empty pattern returns the hierarchy root, which is not a mailbox.
		if pattern == "" {
			for name := range patternMatches {
				roots[name] = struct{}{}
			}
		}

		if matches == nil {
			matches = patternMatches
			continue
		}

		for name, match := range patternMatches {
			if _, ok := matches[name]; !ok {
				matches[name] = match
			}
		}
	}

	if opts.SelectSpecialUse {
		for name, match := range matches {
			if !match.Atts.ContainsAnyUnchecked(specialUseAttributes...) {
				delete(matches, name)
			}
		}
	}

	if opts.ReturnSubscribed {
		subscribed, err := state.subscribedNames(ctx, client, mailboxes)
		if err != nil {
			return nil, err
		}

		for name, match := range matches {
			if _, ok := roots[name]; ok {
				continue
			}

			if _, ok := subscribed[name]; ok {
				match.Atts.AddToSelf(imap.AttrSubscribed)
			}
		}
	}

	return matches, nil
}

func (state *State) listSubscribed(
	ctx context.Context,
	client db.ReadOnly,
	mailboxes []*db.MailboxWithAttr,
	ref string,
	patterns []string,
	opts ListOptions,
) (map[string]Match, error) {
	subscribed, err := state.subscribedNames(ctx, client, mailboxes)
	if err != nil {
		return nil, err
	}

	var (
		matchers = newNameMatchers(ref, patterns, state.delimiter)
		matches  = make(map[string]Match)
		selected = make([]string, 0, len(subscribed))
	)

	for name, mbox := range subscribed {
		if opts.SelectSpecialUse && (mbox == nil || !mbox.Attributes.ContainsAnyUnchecked(specialUseAttributes...)) {
			continue
		}

		selected = append(selected, name)

		if !matchesAnyPattern(matchers, name) {
			continue
		}

		match, err := state.newListMatch(ctx, client, name, mbox)
		if err != nil {
			return nil, err
		}

		match.Atts.AddToSelf(imap.AttrSubscribed)

		matches[name] = match
	}

	if !opts.SelectRecursiveMatch {
		return matches, nil
	}

	var existing map[string]*db.MailboxWithAttr

	for _, name := range selected {
		for superior := range superiors(name, state.delimiter) {
			match, ok := matches[superior]
			if ok && match.ChildInfo != nil {
				continue
			}

			if !ok {
				if !matchesAnyPattern(matchers, superior) {
					continue
				}

				if existing == nil {
					existing = make(map[string]*db.MailboxWithAttr, len(mailboxes))
					for _, mbox := range mailboxes {
						existing[mbox.Name] = mbox
					}
				}

				if match, err = state.newListMatch(ctx, client, superior, existing[superior]); err != nil {
					return nil, err
				}
			}

			match.ChildInfo = []string{ChildInfoSubscribed}

			matches[superior] = match
		}
	}

	return matches, nil
}

// subscribedNames maps a name to nil when it is subscribed but its mailbox no longer exists.
func (state *State) subscribedNames(
	ctx context.Context,
	client db.ReadOnly,
	mailboxes []*db.MailboxWithAttr,
) (map[string]*db.MailboxWithAttr, error) {
	deletedSubscriptions, err := client.GetDeletedSubscriptionSet(ctx)
	if err != nil {
		return nil, err
	}

	subscribed := make(map[string]*db.MailboxWithAttr, len(mailboxes))

	for _, mbox := range mailboxes {
		delete(deletedSubscriptions, mbox.RemoteID)

		if mbox.Subscribed {
			subscribed[mbox.Name] = mbox
		}
	}

	for _, s := range deletedSubscriptions {
		if state.user.GetRemote().GetMailboxVisibility(ctx, s.RemoteID) != imap.Visible {
			continue
		}

		if _, ok := subscribed[s.Name]; !ok {
			subscribed[s.Name] = nil
		}
	}

	return subscribed, nil
}

func (state *State) newListMatch(
	ctx context.Context,
	client db.ReadOnly,
	name string,
	mbox *db.MailboxWithAttr,
) (Match, error) {
	if mbox == nil {
		return Match{
			Name:      name,
			Delimiter: state.delimiter,
			Atts:      imap.NewFlagSet(imap.AttrNonExistent),
		}, nil
	}

	match, _, err := prepareMatch(
		ctx, client, name, &matchMailbox{Name: name, EntMBox: mbox},
		"", state.delimiter,
		true, true, false,
	)

	return match, err
}

func newNameMatchers(ref string, patterns []string, delimiter string) []nameMatcher {
	matchers := make([]nameMatcher, 0, len(patterns))

	for _, pattern := range patterns {
		if pattern != "" {
			matchers = append(matchers, newMatcher(ref, pattern, delimiter))
		}
	}

	return matchers
}

// matchesAnyPattern is true only when the whole name matches, not just one of its superiors.
func matchesAnyPattern(matchers []nameMatcher, name string) bool {
	for _, matchName := range matchers {
		if matched, ok := matchName(name); ok && len(matched) == len(name) {
			return true
		}
	}

	return false
}

// addChildAttributes follows RFC 3348. A \Noinferiors mailbox gets neither attribute.
func addChildAttributes(
	matches map[string]Match,
	mailboxes []*db.MailboxWithAttr,
	delimiter string,
	skip map[string]struct{},
) {
	var parents map[string]struct{}

	for _, mbox := range mailboxes {
		for superior := range superiors(mbox.Name, delimiter) {
			if _, ok := matches[superior]; !ok {
				continue
			}

			if parents == nil {
				parents = make(map[string]struct{})
			}

			parents[superior] = struct{}{}
		}
	}

	for name, match := range matches {
		if _, ok := skip[name]; ok || name == "" {
			continue
		}

		if match.Atts.ContainsUnchecked(noInferiorsAttribute) {
			continue
		}

		if _, ok := parents[name]; ok {
			match.Atts.AddToSelf(imap.AttrHasChildren)
		} else {
			match.Atts.AddToSelf(imap.AttrHasNoChildren)
		}
	}
}

// superiors yields every name above the given one in the hierarchy, without allocating.
func superiors(name, delimiter string) iter.Seq[string] {
	return func(yield func(string) bool) {
		if delimiter == "" {
			return
		}

		for end := strings.Index(name, delimiter); end >= 0; {
			if !yield(name[:end]) {
				return
			}

			next := strings.Index(name[end+len(delimiter):], delimiter)
			if next < 0 {
				return
			}

			end += len(delimiter) + next
		}
	}
}
