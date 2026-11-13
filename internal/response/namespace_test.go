package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamespace(t *testing.T) {
	assert.Equal(t, `* NAMESPACE (("" "/")) NIL NIL`, Namespace("/").String())
}

func TestNamespaceDotDelimiter(t *testing.T) {
	assert.Equal(t, `* NAMESPACE (("" ".")) NIL NIL`, Namespace(".").String())
}

func TestNamespaceBackslashDelimiterIsEscaped(t *testing.T) {
	assert.Equal(t, `* NAMESPACE (("" "\\")) NIL NIL`, Namespace(`\`).String())
}

func TestNamespaceNilDelimiter(t *testing.T) {
	assert.Equal(t, `* NAMESPACE (("" NIL)) NIL NIL`, Namespace("").String())
}
