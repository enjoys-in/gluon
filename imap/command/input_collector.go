package command

import (
	"io"
	"slices"

	"github.com/ProtonMail/gluon/rfcparser"
)

const maxRetainedInputSize = 64 * 1024

type InputCollector struct {
	source rfcparser.Reader
	bytes  []byte
}

func NewInputCollector(source rfcparser.Reader) *InputCollector {
	return &InputCollector{
		source: source,
		bytes:  make([]byte, 0, 128),
	}
}

func (i *InputCollector) Bytes() []byte {
	return i.bytes
}

func (i *InputCollector) Read(dst []byte) (int, error) {
	n, err := i.source.Read(dst)
	if err == nil {
		i.bytes = append(i.bytes, dst[0:n]...)
	}

	return n, err
}

func (i *InputCollector) ReadByte() (byte, error) {
	b, err := i.source.ReadByte()
	if err == nil {
		i.bytes = append(i.bytes, b)
	}

	return b, err
}

func (i *InputCollector) ReadBytes(delim byte) ([]byte, error) {
	b, err := i.source.ReadBytes(delim)
	if err == nil {
		i.bytes = append(i.bytes, b...)
	}

	return b, err
}

func (i *InputCollector) Discard(n int) error {
	_, err := io.CopyN(io.Discard, i.source, int64(n))

	return err
}

func (i *InputCollector) Grow(n int) {
	i.bytes = slices.Grow(i.bytes, n)
}

func (i *InputCollector) Reset() {
	// A buffer that grew for a large literal is reused by the next large command and released by the next small one.
	if cap(i.bytes) > maxRetainedInputSize && len(i.bytes) < cap(i.bytes)/4 {
		i.bytes = make([]byte, 0, 128)

		return
	}

	i.bytes = i.bytes[:0]
}

func (i *InputCollector) SetSource(source rfcparser.Reader) {
	i.source = source
}
