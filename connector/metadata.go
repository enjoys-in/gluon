package connector

import (
	"errors"
	"fmt"
)

// The errors with which SetMetadata refuses an annotation (RFC 5464).
var (
	ErrMetadataTooMany   = errors.New("too many annotations")
	ErrMetadataNoPrivate = errors.New("private annotations are not supported")
)

// MetadataTooBigError refuses an annotation whose value is larger than MaxSize bytes, the largest the connector
// accepts.
type MetadataTooBigError struct {
	MaxSize int
}

func (err *MetadataTooBigError) Error() string {
	return fmt.Sprintf("annotation is larger than %v bytes", err.MaxSize)
}
