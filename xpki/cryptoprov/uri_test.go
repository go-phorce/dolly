package cryptoprov_test

import (
	"testing"

	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseTokenURI(t *testing.T) {
	c, err := cryptoprov.ParseTokenURI("pkcs11:manufacturer=test;model=inmem;serial=20764350726;token=inmemoryRSA")
	require.NoError(t, err)

	assert.Equal(t, "test", c.Manufacturer())
	assert.Equal(t, "inmem", c.Model())
	assert.Equal(t, "20764350726", c.TokenSerial())
	assert.Equal(t, "inmemoryRSA", c.TokenLabel())
}

func Test_ParsePrivateKeyURI(t *testing.T) {
	uri, err := cryptoprov.ParsePrivateKeyURI("pkcs11:manufacturer=test;model=inmem;serial=20764350726;token=inmemoryRSA;id=123;type=private")
	require.NoError(t, err)

	assert.Equal(t, "123", uri.ID())
	assert.Equal(t, "test", uri.Manufacturer())
	assert.Equal(t, "inmem", uri.Model())
	assert.Equal(t, "20764350726", uri.TokenSerial())
	assert.Equal(t, "inmemoryRSA", uri.TokenLabel())
}
