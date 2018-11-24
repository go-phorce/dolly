package tags_test

import (
	"testing"

	"github.com/go-phorce/dolly/metrics/tags"
	"github.com/stretchr/testify/assert"
)

func Test_Tags(t *testing.T) {
	assert.Equal(t, "uri", tags.URI)
	assert.Equal(t, "method", tags.Method)
	assert.Equal(t, "status", tags.Status)
	assert.Equal(t, "role", tags.Role)
}
