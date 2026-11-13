package response

import (
	"fmt"
	"strings"
)

// ThreadNode represents a node in the thread tree.
type ThreadNode struct {
	Num      uint32
	Children []*ThreadNode
}

type threadResponse struct {
	roots []*ThreadNode
}

// Thread creates a THREAD untagged response: * THREAD (thread1)(thread2)...
func Thread(roots []*ThreadNode) *threadResponse {
	return &threadResponse{
		roots: roots,
	}
}

func (r *threadResponse) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *threadResponse) String() string {
	if len(r.roots) == 0 {
		return "* THREAD"
	}

	var sb strings.Builder

	sb.WriteString("* THREAD ")

	for _, root := range r.roots {
		writeThreadNode(&sb, root, true)
	}

	return sb.String()
}

// writeThreadNode serializes a thread node into RFC 5256 format.
// The format is: (num (child1)(child2)) for branching, or (num child) for linear chains.
func writeThreadNode(sb *strings.Builder, node *ThreadNode, isRoot bool) {
	if isRoot {
		sb.WriteString("(")
	}

	if node.Num > 0 {
		fmt.Fprintf(sb, "%d", node.Num)
	}

	if node.Num > 0 && len(node.Children) > 0 {
		sb.WriteString(" ")
	}

	if len(node.Children) == 1 {
		// Linear chain: parent child
		writeThreadNode(sb, node.Children[0], false)
	} else if len(node.Children) > 1 {
		// Branching: parent (child1)(child2)
		for _, child := range node.Children {
			writeThreadNode(sb, child, true)
		}
	}

	if isRoot {
		sb.WriteString(")")
	}
}
