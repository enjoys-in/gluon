package response

import (
	"bytes"
	"strconv"
	"strings"
)

type itemBinary struct {
	part    []int
	content []byte
	partial int
}

// ItemBinary is the BINARY fetch item (RFC 3516): a body part without its content transfer encoding.
func ItemBinary(part []int, content []byte) *itemBinary {
	return &itemBinary{
		part:    part,
		content: content,
		partial: -1,
	}
}

func (r *itemBinary) WithPartial(begin, count int) *itemBinary {
	r.partial = begin

	if contentLen := len(r.content); begin >= contentLen {
		r.content = nil
	} else if begin+count > contentLen {
		r.content = r.content[begin:]
	} else {
		r.content = r.content[begin : begin+count]
	}

	return r
}

func (r *itemBinary) String() string {
	var partial string

	if r.partial >= 0 {
		partial = "<" + strconv.Itoa(r.partial) + ">"
	}

	// Content with a zero byte is sent as a binary literal, other content as a plain one.
	var binary string

	if bytes.IndexByte(r.content, 0) >= 0 {
		binary = "~"
	}

	return "BINARY[" + binarySection(r.part) + "]" + partial + " " + binary + "{" + strconv.Itoa(len(r.content)) + "}\r\n" + string(r.content)
}

type itemBinarySize struct {
	part []int
	size int
}

// ItemBinarySize is the BINARY.SIZE fetch item (RFC 3516).
func ItemBinarySize(part []int, size int) *itemBinarySize {
	return &itemBinarySize{
		part: part,
		size: size,
	}
}

func (r *itemBinarySize) String() string {
	return "BINARY.SIZE[" + binarySection(r.part) + "] " + strconv.Itoa(r.size)
}

func binarySection(part []int) string {
	numbers := make([]string, 0, len(part))

	for _, number := range part {
		numbers = append(numbers, strconv.Itoa(number))
	}

	return strings.Join(numbers, ".")
}

type itemUnknownCTE struct{}

// ItemUnknownCTE tells a client that a body part has a content transfer encoding the server cannot remove.
func ItemUnknownCTE() *itemUnknownCTE {
	return &itemUnknownCTE{}
}

func (c *itemUnknownCTE) String() string {
	return "UNKNOWN-CTE"
}
