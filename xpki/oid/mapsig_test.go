package oid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SignatureAlgorithmByOID(t *testing.T) {
	for _, c := range oidTests {
		t.Run(c.name, func(t *testing.T) {
			oi := LookupByOID(c.oid)
			assert.NotNil(t, oi, "LookupByOID %s")

			hi, err := SignatureAlgorithmByOID(oi.String())
			if oi.Type() == AlgSig {
				assert.NoError(t, err)
				assert.NotNil(t, hi)
			} else {
				assert.Error(t, err)
				assert.Nil(t, hi)

			}
		})
	}
}
