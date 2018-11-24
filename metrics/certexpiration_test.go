package metrics

import (
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	gm "github.com/armon/go-metrics"
	"github.com/go-phorce/dolly/testify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PublishCertExpiration(t *testing.T) {
	im := gm.NewInmemSink(time.Minute, time.Minute*5)
	_, err := gm.NewGlobal(gm.DefaultConfig("service"), im)
	require.NoError(t, err)
	SetProvider(NewStandardProvider())

	crt, _, err := testify.MakeSelfCertRSA(24)
	require.NoError(t, err)
	if crt.SubjectKeyId == nil {
		crt.SubjectKeyId = []byte("123")
	}

	expiresInDays := PublishCertExpirationInDays(crt, "longlived")
	assert.True(t, expiresInDays > 0)
	assert.True(t, expiresInDays <= 1)

	expiresInDays = PublishShortLivedCertExpirationInDays(crt, "shortlived")
	assert.True(t, expiresInDays > 0)
	assert.True(t, expiresInDays <= 1)

	crl := &pkix.CertificateList{
		TBSCertList: pkix.TBSCertificateList{
			NextUpdate: crt.NotAfter,
		},
	}
	expiresInDays = PublishCRLExpirationInDays(crl, crt)
	assert.True(t, expiresInDays > 0)
	assert.True(t, expiresInDays <= 1)

	// get samples in memory
	data := im.Data()
	require.NotEqual(t, 0, len(data))

	for k := range data[0].Gauges {
		t.Log("Gauge:", k)
	}

	assertGauge := func(key string) {
		s, exists := data[0].Gauges[key]
		require.True(t, exists, "Expected metric with key %s to exist, but it doesn't", key)
		assert.True(t, s.Value > 0 && s.Value <= 1, "Unexpected value for metric %s", key)
	}
	hostname, _ := os.Hostname()
	assertGauge(
		fmt.Sprintf("service.%s.cert.expiry.days;CN=%s;type=longlived;Serial=%s;SKI=%s",
			hostname, crt.Subject.CommonName, crt.SerialNumber.String(), hex.EncodeToString(crt.SubjectKeyId)))
	assertGauge(
		fmt.Sprintf("service.%s.cert.expiry.days;CN=%s;type=shortlived",
			hostname, crt.Subject.CommonName))
	assertGauge(
		fmt.Sprintf("service.%s.cert.expiry.days;CN=%s;type=issuer;Serial=%s;SKI=%s",
			hostname, crt.Subject.CommonName, crt.SerialNumber.String(), hex.EncodeToString(crt.SubjectKeyId)))
	assertGauge(
		fmt.Sprintf("service.%s.crl.expiry.days;CN=%s;Serial=%s;SKI=%s",
			hostname, crt.Subject.CommonName, crt.SerialNumber.String(), hex.EncodeToString(crt.SubjectKeyId)))
}
