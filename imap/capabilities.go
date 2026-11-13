package imap

type Capability string

const (
	IMAP4rev1        Capability = `IMAP4rev1`
	StartTLS         Capability = `STARTTLS`
	IDLE             Capability = `IDLE`
	UNSELECT         Capability = `UNSELECT`
	UIDPLUS          Capability = `UIDPLUS`
	MOVE             Capability = `MOVE`
	ID               Capability = `ID`
	AUTHPLAIN        Capability = `AUTH=PLAIN`
	QUOTA            Capability = `QUOTA`
	SORT             Capability = `SORT`
	ThreadReferences Capability = `THREAD=REFERENCES`
	NAMESPACE        Capability = `NAMESPACE`
	CHILDREN         Capability = `CHILDREN`
	SpecialUse       Capability = `SPECIAL-USE`
	ListExtended     Capability = `LIST-EXTENDED`
	ListStatus       Capability = `LIST-STATUS`
	ESEARCH          Capability = `ESEARCH`
	LiteralPlus      Capability = `LITERAL+`
	SASLIR           Capability = `SASL-IR`
	UNAUTHENTICATE   Capability = `UNAUTHENTICATE`
	StatusSize       Capability = `STATUS=SIZE`
	WITHIN           Capability = `WITHIN`
	ESORT            Capability = `ESORT`
	SortDisplay      Capability = `SORT=DISPLAY`
	SEARCHRES        Capability = `SEARCHRES`
)

func IsCapabilityAvailableBeforeAuth(c Capability) bool {
	switch c {
	case IMAP4rev1, StartTLS, IDLE, ID, AUTHPLAIN, LiteralPlus, SASLIR:
		return true
	case UNSELECT, UIDPLUS, MOVE, QUOTA, SORT, ThreadReferences, NAMESPACE,
		CHILDREN, SpecialUse, ListExtended, ListStatus, ESEARCH, UNAUTHENTICATE, StatusSize, WITHIN, ESORT, SortDisplay, SEARCHRES:
		return false
	}

	return false
}
