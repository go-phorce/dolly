package crypto11

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GenerateKeyID(t *testing.T) {
	id, err := p11lib.generateKeyID()
	require.NoError(t, err)
	assert.Equal(t, 32, len(id))
}

func Test_GenerateKeyLabel(t *testing.T) {
	label, err := p11lib.generateKeyLabel()
	require.NoError(t, err)
	assert.Equal(t, 32, len(label))

	now := time.Now().UTC()
	prefix := fmt.Sprintf("%04d%02d%02d%02d%02d%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	assert.True(t, strings.HasPrefix(string(label), prefix), "Key should have prefix %q, got %q", prefix, label)
}
