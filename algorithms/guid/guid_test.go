package guid_test

import (
	"testing"

	"github.com/go-phorce/pkg/algorithms/guid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Guid(t *testing.T) {
	g := guid.MustCreate()
	require.NotEmpty(t, g)
	assert.Equal(t, 36, len(g))
}
