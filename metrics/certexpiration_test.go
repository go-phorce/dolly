package metrics

import (
	"testing"

	"github.com/go-phorce/pkg/testify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PublishCertExpiration(t *testing.T) {
	crt, _, err := testify.MakeSelfCertRSA(24)
	require.NoError(t, err)

	expiresInDays := PublishCertExpirationInDays(crt, "test")
	assert.True(t, expiresInDays > 0)
	assert.True(t, expiresInDays <= 1)
}
