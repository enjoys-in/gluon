package state

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/ProtonMail/gluon/async"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/contexts"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/rfc5322"
	"github.com/ProtonMail/gluon/rfc822"
	"github.com/bradenaw/juniper/parallel"
	"golang.org/x/text/encoding"
)

// threadContainer is a container used in the JWZ threading algorithm.
type threadContainer struct {
	msgInfo *threadMessageInfo
	parent  *threadContainer
	child   *threadContainer // first child
	next    *threadContainer // next sibling
}

// threadMessageInfo holds information about a single message for threading.
type threadMessageInfo struct {
	messageID  string
	references []string
	subject    string
	date       string
	seq        imap.SeqID
	uid        imap.UID
}

// Thread performs the THREAD command per RFC 5256.
func (m *Mailbox) Thread(ctx context.Context, algorithm string, searchKeys []command.SearchKey, decoder *encoding.Decoder) ([]*response.ThreadNode, error) {
	// Filter messages using existing search infrastructure.
	op, err := buildSearchOpListWithKeys(m, searchKeys, decoder)
	if err != nil {
		return nil, err
	}

	msgCount := m.snap.len()

	type matchedMsg struct {
		msg   snapMsgWithSeq
		match bool
	}

	matched := make([]matchedMsg, msgCount)

	activeRequests := atomic.AddInt32(&totalActiveSearchRequests, 1)
	defer atomic.AddInt32(&totalActiveSearchRequests, -1)

	var parallelismCount int

	if contexts.IsParallelismDisabledCtx(ctx) {
		parallelismCount = 1
	} else {
		parallelismCount = runtime.NumCPU() / int(activeRequests)
	}

	if parallelismCount < 1 {
		parallelismCount = 1
	}

	if err := parallel.DoContext(ctx, parallelismCount, msgCount, func(ctx context.Context, i int) error {
		defer async.HandlePanic(m.state.panicHandler)

		msg, ok := m.snap.messages.getWithSeqID(imap.SeqID(uint32(i + 1)))
		if !ok {
			return nil
		}

		matches, err := applySearch(ctx, m, msg, op)
		if err != nil {
			return err
		}

		matched[i] = matchedMsg{msg: msg, match: matches}

		return nil
	}); err != nil {
		return nil, err
	}

	// Collect matched messages.
	var matchedMsgs []snapMsgWithSeq

	for _, m := range matched {
		if m.match {
			matchedMsgs = append(matchedMsgs, m.msg)
		}
	}

	if len(matchedMsgs) == 0 {
		return nil, nil
	}

	// Extract threading info (Message-ID, References, In-Reply-To, Subject, Date).
	infos := make([]*threadMessageInfo, len(matchedMsgs))

	if err := parallel.DoContext(ctx, parallelismCount, len(matchedMsgs), func(ctx context.Context, i int) error {
		defer async.HandlePanic(m.state.panicHandler)

		msg := matchedMsgs[i]

		literal, err := m.state.getLiteral(ctx, msg.ID)
		if err != nil {
			return err
		}

		headerBytes, _ := rfc822.Split(literal)

		header, err := rfc822.NewHeader(headerBytes)
		if err != nil {
			return err
		}

		info := &threadMessageInfo{
			messageID: extractMessageID(header.Get("Message-Id")),
			subject:   header.Get("Subject"),
			seq:       msg.Seq,
			uid:       msg.UID,
		}

		// Collect references.
		refs := parseMessageIDList(header.Get("References"))

		inReplyTo := extractMessageID(header.Get("In-Reply-To"))
		if inReplyTo != "" {
			// Add In-Reply-To if not already in References.
			found := false
			for _, r := range refs {
				if r == inReplyTo {
					found = true
					break
				}
			}

			if !found {
				refs = append(refs, inReplyTo)
			}
		}

		info.references = refs

		dateStr := header.Get("Date")
		if dt, err := rfc5322.ParseDateTime(dateStr); err == nil {
			info.date = dt.UTC().Format("20060102150405")
		}

		infos[i] = info

		return nil
	}); err != nil {
		return nil, err
	}

	if algorithm == "ORDEREDSUBJECT" {
		return threadByOrderedSubject(infos, contexts.IsUID(ctx)), nil
	}

	return threadByReferences(infos, contexts.IsUID(ctx)), nil
}

// threadByReferences implements the REFERENCES threading algorithm per RFC 5256 §3.
func threadByReferences(infos []*threadMessageInfo, useUID bool) []*response.ThreadNode {
	idTable := make(map[string]*threadContainer)

	// Step 1: For each message, create or find a container and link parent/child.
	for _, info := range infos {
		if info == nil {
			continue
		}

		// Ensure a container exists for this message's Message-ID.
		msgID := info.messageID
		if msgID == "" {
			// Generate a unique placeholder.
			msgID = generatePlaceholderID(info)
		}

		container := getOrCreateContainer(idTable, msgID)
		container.msgInfo = info

		// Walk through References to build the chain.
		var parentContainer *threadContainer

		for _, ref := range info.references {
			refContainer := getOrCreateContainer(idTable, ref)

			// Link parent -> child if no existing parent and no loop.
			if parentContainer != nil && refContainer.parent == nil && !isDescendant(refContainer, parentContainer) {
				addChild(parentContainer, refContainer)
			}

			parentContainer = refContainer
		}

		// Link the last reference as parent of this message.
		if parentContainer != nil && parentContainer != container && !isDescendant(container, parentContainer) {
			// Remove from old parent if any.
			if container.parent != nil {
				removeChild(container.parent, container)
			}

			addChild(parentContainer, container)
		}
	}

	// Step 2: Find root set (containers with no parent).
	var roots []*threadContainer

	for _, c := range idTable {
		if c.parent == nil {
			roots = append(roots, c)
		}
	}

	// Step 3: Prune empty containers.
	roots = pruneEmptyContainers(roots)

	// Step 4: Sort siblings by date.
	sortSiblingsByDate(roots)

	// Step 5: Group by base subject (simplified).
	roots = groupBySubject(roots)

	// Sort root set by date of first message.
	sortContainersByDate(roots)

	// Convert to response ThreadNodes.
	return convertToThreadNodes(roots, useUID)
}

// threadByOrderedSubject implements the ORDEREDSUBJECT algorithm per RFC 5256 §2.
func threadByOrderedSubject(infos []*threadMessageInfo, useUID bool) []*response.ThreadNode {
	// Group by base subject.
	groups := make(map[string][]*threadMessageInfo)

	for _, info := range infos {
		if info == nil {
			continue
		}

		base := extractBaseSubject(info.subject)
		groups[base] = append(groups[base], info)
	}

	var roots []*response.ThreadNode

	for _, group := range groups {
		// Sort within group by date, then by sequence number.
		sortInfosByDate(group)

		if len(group) == 1 {
			roots = append(roots, &response.ThreadNode{
				Num: getNum(group[0], useUID),
			})
		} else {
			// First message is the parent, rest are children.
			root := &response.ThreadNode{
				Num: getNum(group[0], useUID),
			}

			for _, child := range group[1:] {
				root.Children = append(root.Children, &response.ThreadNode{
					Num: getNum(child, useUID),
				})
			}

			roots = append(roots, root)
		}
	}

	// Sort roots by the date of their first message.
	sortThreadNodesByFirstMsg(roots, infos, useUID)

	return roots
}

func getNum(info *threadMessageInfo, useUID bool) uint32 {
	if useUID {
		return uint32(info.uid)
	}

	return uint32(info.seq)
}

func getOrCreateContainer(table map[string]*threadContainer, id string) *threadContainer {
	if c, ok := table[id]; ok {
		return c
	}

	c := &threadContainer{}
	table[id] = c

	return c
}

func addChild(parent, child *threadContainer) {
	child.parent = parent
	child.next = parent.child
	parent.child = child
}

func removeChild(parent, child *threadContainer) {
	if parent.child == child {
		parent.child = child.next
		child.parent = nil
		child.next = nil

		return
	}

	prev := parent.child
	for prev != nil && prev.next != child {
		prev = prev.next
	}

	if prev != nil {
		prev.next = child.next
	}

	child.parent = nil
	child.next = nil
}

func isDescendant(container, possibleAncestor *threadContainer) bool {
	for c := possibleAncestor.parent; c != nil; c = c.parent {
		if c == container {
			return true
		}
	}

	return false
}

func pruneEmptyContainers(roots []*threadContainer) []*threadContainer {
	var result []*threadContainer

	for _, root := range roots {
		pruned := pruneContainer(root)
		result = append(result, pruned...)
	}

	return result
}

func pruneContainer(c *threadContainer) []*threadContainer {
	// Recursively prune children first.
	var newChildren []*threadContainer

	child := c.child
	for child != nil {
		nextChild := child.next
		child.next = nil

		pruned := pruneContainer(child)
		newChildren = append(newChildren, pruned...)

		child = nextChild
	}

	// Rebuild child list.
	c.child = nil

	for i := len(newChildren) - 1; i >= 0; i-- {
		newChildren[i].parent = c
		newChildren[i].next = c.child
		c.child = newChildren[i]
	}

	// If this container is empty (no message):
	if c.msgInfo == nil {
		if c.child == nil {
			// No children — discard entirely.
			return nil
		}

		// Promote children to replace this container.
		var promoted []*threadContainer

		ch := c.child
		for ch != nil {
			next := ch.next
			ch.parent = nil
			ch.next = nil
			promoted = append(promoted, ch)

			ch = next
		}

		return promoted
	}

	return []*threadContainer{c}
}

func getContainerDate(c *threadContainer) string {
	if c.msgInfo != nil {
		return c.msgInfo.date
	}

	// Use the earliest child date.
	earliest := ""

	child := c.child
	for child != nil {
		d := getContainerDate(child)
		if d != "" && (earliest == "" || d < earliest) {
			earliest = d
		}

		child = child.next
	}

	return earliest
}

func sortSiblingsByDate(containers []*threadContainer) {
	for _, c := range containers {
		if c.child != nil {
			children := collectChildren(c)
			sortContainersByDate(children)
			rebuildChildList(c, children)

			sortSiblingsByDate(children)
		}
	}
}

func collectChildren(parent *threadContainer) []*threadContainer {
	var children []*threadContainer

	child := parent.child
	for child != nil {
		children = append(children, child)
		child = child.next
	}

	return children
}

func rebuildChildList(parent *threadContainer, children []*threadContainer) {
	parent.child = nil

	for i := len(children) - 1; i >= 0; i-- {
		children[i].next = parent.child
		children[i].parent = parent
		parent.child = children[i]
	}
}

func sortContainersByDate(containers []*threadContainer) {
	for i := 1; i < len(containers); i++ {
		for j := i; j > 0; j-- {
			di := getContainerDate(containers[j])
			dj := getContainerDate(containers[j-1])

			if di < dj {
				containers[j], containers[j-1] = containers[j-1], containers[j]
			} else {
				break
			}
		}
	}
}

func groupBySubject(roots []*threadContainer) []*threadContainer {
	// Build subject table.
	subjectTable := make(map[string]*threadContainer)

	for _, root := range roots {
		subject := getContainerSubject(root)
		base := extractBaseSubject(subject)

		if base == "" {
			continue
		}

		existing, ok := subjectTable[base]
		if !ok {
			subjectTable[base] = root
			continue
		}

		// Keep the one that is more likely the real root (the one without Re:).
		existingIsReply := isReply(getContainerSubject(existing))
		currentIsReply := isReply(subject)

		if existingIsReply && !currentIsReply {
			subjectTable[base] = root
		} else if existing.msgInfo == nil && root.msgInfo != nil {
			subjectTable[base] = root
		}
	}

	// Merge threads with same subject.
	var result []*threadContainer
	merged := make(map[*threadContainer]bool)

	for _, root := range roots {
		if merged[root] {
			continue
		}

		subject := getContainerSubject(root)
		base := extractBaseSubject(subject)

		if base == "" {
			result = append(result, root)
			continue
		}

		target, ok := subjectTable[base]
		if !ok || target == root {
			result = append(result, root)
			continue
		}

		// Merge root into target.
		if target.msgInfo == nil {
			// Target is empty container — just add root as child.
			addChild(target, root)
		} else if root.msgInfo == nil {
			// Root is empty — add target as child of root, swap.
			addChild(root, target)
			subjectTable[base] = root

			result = append(result, root)
			merged[target] = true

			continue
		} else {
			// Both have messages — make root a child of target.
			addChild(target, root)
		}

		merged[root] = true
	}

	return result
}

func getContainerSubject(c *threadContainer) string {
	if c.msgInfo != nil {
		return c.msgInfo.subject
	}

	child := c.child
	for child != nil {
		if child.msgInfo != nil {
			return child.msgInfo.subject
		}

		child = child.next
	}

	return ""
}

var replyPrefixPattern = regexp.MustCompile(`(?i)^(re|fwd|fw)\s*(\[\d+\])?\s*:`)

func isReply(subject string) bool {
	subject = strings.TrimSpace(subject)
	return replyPrefixPattern.MatchString(subject)
}

func convertToThreadNodes(containers []*threadContainer, useUID bool) []*response.ThreadNode {
	var nodes []*response.ThreadNode

	for _, c := range containers {
		node := containerToThreadNode(c, useUID)
		if node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func containerToThreadNode(c *threadContainer, useUID bool) *response.ThreadNode {
	if c == nil {
		return nil
	}

	node := &response.ThreadNode{}

	if c.msgInfo != nil {
		node.Num = getNum(c.msgInfo, useUID)
	}

	child := c.child
	for child != nil {
		childNode := containerToThreadNode(child, useUID)
		if childNode != nil {
			node.Children = append(node.Children, childNode)
		}

		child = child.next
	}

	return node
}

var messageIDPattern = regexp.MustCompile(`<([^>]+)>`)

func extractMessageID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	matches := messageIDPattern.FindStringSubmatch(value)
	if len(matches) >= 2 {
		return matches[1]
	}

	return value
}

func parseMessageIDList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	matches := messageIDPattern.FindAllStringSubmatch(value, -1)

	var result []string

	for _, m := range matches {
		if len(m) >= 2 {
			result = append(result, m[1])
		}
	}

	return result
}

func generatePlaceholderID(info *threadMessageInfo) string {
	return fmt.Sprintf("placeholder-%d-%d", info.seq, info.uid)
}

func sortInfosByDate(infos []*threadMessageInfo) {
	for i := 1; i < len(infos); i++ {
		for j := i; j > 0; j-- {
			if infos[j].date < infos[j-1].date {
				infos[j], infos[j-1] = infos[j-1], infos[j]
			} else {
				break
			}
		}
	}
}

func sortThreadNodesByFirstMsg(nodes []*response.ThreadNode, infos []*threadMessageInfo, useUID bool) {
	// Build a map from num -> date for sorting.
	dateMap := make(map[uint32]string)

	for _, info := range infos {
		if info != nil {
			num := getNum(info, useUID)
			dateMap[num] = info.date
		}
	}

	for i := 1; i < len(nodes); i++ {
		for j := i; j > 0; j-- {
			di := dateMap[nodes[j].Num]
			dj := dateMap[nodes[j-1].Num]

			if di < dj {
				nodes[j], nodes[j-1] = nodes[j-1], nodes[j]
			} else {
				break
			}
		}
	}
}
