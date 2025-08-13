package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_cacheId_getElement(t *testing.T) {
	id := newCacheId("identity", "uri/a/b", true)
	assert.Equal(t, "identity", id.Identity())
	assert.Equal(t, "uri/a/b", id.Uri())
	assert.True(t, id.WithName())
	assert.Equal(t, "{identity}/uri/a/b/1", id.String())
}
