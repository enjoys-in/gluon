package imap

const (
	AttrNoSelect    = `\Noselect`
	AttrNoInferiors = `\Noinferiors`
	AttrMarked      = `\Marked`
	AttrUnmarked    = `\Unmarked`

	// Child attributes as defined in RFC-3348.
	AttrHasChildren   = `\HasChildren`
	AttrHasNoChildren = `\HasNoChildren`

	// Extended LIST attributes as defined in RFC-5258.
	AttrSubscribed  = `\Subscribed`
	AttrNonExistent = `\NonExistent`

	// Special Use attributes as defined in RFC-6154.
	AttrAll     = `\All`
	AttrArchive = `\Archive`
	AttrDrafts  = `\Drafts`
	AttrFlagged = `\Flagged`
	AttrJunk    = `\Junk`
	AttrSent    = `\Sent`
	AttrTrash   = `\Trash`
)

// SpecialUseAttributes lists the RFC-6154 attributes that mark a mailbox as having a special use.
var SpecialUseAttributes = []string{AttrAll, AttrArchive, AttrDrafts, AttrFlagged, AttrJunk, AttrSent, AttrTrash}
