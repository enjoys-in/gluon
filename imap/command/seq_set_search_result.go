package command

// SeqNumValueSearchResult stands for "$", the saved result of the last SEARCH RETURN (SAVE) (RFC 5182).
const SeqNumValueSearchResult = SeqNum(-1)

func (s SeqNum) IsSearchResult() bool {
	return s == SeqNumValueSearchResult
}

// IsSearchResultSeqSet reports whether the sequence set is "$".
func IsSearchResultSeqSet(seqSet []SeqRange) bool {
	return len(seqSet) == 1 && seqSet[0].Begin.IsSearchResult()
}
