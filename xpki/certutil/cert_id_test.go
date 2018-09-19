package certutil_test

import (
	"testing"

	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_IDs(t *testing.T) {
	crt, err := certutil.ParseFromPEM([]byte(GoDaddyIntermediateCert))
	require.NoError(t, err)

	str, err := certutil.GetThumbprintStr(crt)
	require.NoError(t, err)
	assert.Equal(t, "7c4656c3061f7f4c0d67b319a855f60ebc11fc44", str)

	str = certutil.GetSubjectKeyID(crt)
	assert.Equal(t, "fdac6132936c45d6e2ee855f9abae7769968cce7", str)

	str = certutil.GetAuthorityKeyID(crt)
	assert.Equal(t, "d2c4b0d291d44c1171b361cb3da1fedda86ad4e3", str)

	str = certutil.GetSubjectID(crt)
	assert.Equal(t, "fdac6132936c45d6e2ee855f9abae7769968cce7", str)

	str = certutil.GetIssuerID(crt)
	assert.Equal(t, "d2c4b0d291d44c1171b361cb3da1fedda86ad4e3", str)
}
