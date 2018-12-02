package gpg

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/go-phorce/dolly/testify"
	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/openpgp/packet"
)

type sha256Test struct {
	out string
	in  string
}

var golden256 = []sha256Test{
	{"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", ""},
	{"ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", "a"},
	{"fb8e20fc2e4c3f248c60c39bd652f3c1347298bb977b8b4d5903b85055620603", "ab"},
	{"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", "abc"},
	{"88d4266fd4e6338d13b845fcf289579d209c897823b9217da3e161936f031589", "abcd"},
	{"36bbe50ed96841d10443bcb670d6554f0a34b761be67ec9c4a8ad2c0c44ca42c", "abcde"},
	{"bef57ec7f53a6d40beb640a780a639c83bc29ac8a9816f1fc6c5c6dcd93c4721", "abcdef"},
	{"7d1a54127b222502f5b79b5fb0803061152a44f92b37e23c6527baf665d4da9a", "abcdefg"},
	{"9c56cc51b374c3ba189210d5b6d4bf57790d351c96c47c02190ecf1e430635ab", "abcdefgh"},
	{"19cc02f26df43cc571bc9ed7b0c4d29224a3ec229529221725ef76d021c8326f", "abcdefghi"},
	{"72399361da6a7754fec986dca5b7cbaf1c810a28ded4abaf56b2106d06cb78b0", "abcdefghij"},
	{"a144061c271f152da4d151034508fed1c138b8c976339de229c3bb6d4bbb4fce", "Discard medicine more than two years old."},
	{"6dae5caa713a10ad04b46028bf6dad68837c581616a1589a265a11288d4bb5c4", "He who has a shady past knows that nice guys finish last."},
	{"ae7a702a9509039ddbf29f0765e70d0001177914b86459284dab8b348c2dce3f", "I wouldn't marry him with a ten foot pole."},
	{"6748450b01c568586715291dfa3ee018da07d36bb7ea6f180c1af6270215c64f", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave"},
	{"14b82014ad2b11f661b5ae6a99b75105c2ffac278cd071cd6c05832793635774", "The days of the digital watch are numbered.  -Tom Stoppard"},
	{"7102cfd76e2e324889eece5d6c41921b1e142a4ac5a2692be78803097f6a48d8", "Nepal premier won't resign."},
	{"23b1018cd81db1d67983c5f7417c44da9deb582459e378d7a068552ea649dc9f", "For every action there is an equal and opposite government program."},
	{"8001f190dfb527261c4cfcab70c98e8097a7a1922129bc4096950e57c7999a5a", "His money is twice tainted: 'taint yours and 'taint mine."},
	{"8c87deb65505c3993eb24b7a150c4155e82eee6960cf0c3a8114ff736d69cad5", "There is no reason for any individual to have a computer in their home. -Ken Olsen, 1977"},
	{"bfb0a67a19cdec3646498b2e0f751bddc41bba4b7f30081b0b932aad214d16d7", "It's a tiny change to the code and not completely disgusting. - Bob Manchek"},
	{"7f9a0b9bf56332e19f5a0ec1ad9c1425a153da1c624868fda44561d6b74daf36", "size:  a.out:  bad magic"},
	{"b13f81b8aad9e3666879af19886140904f7f429ef083286195982a7588858cfc", "The major problem is with sendmail.  -Mark Horton"},
	{"b26c38d61519e894480c70c8374ea35aa0ad05b2ae3d6674eec5f52a69305ed4", "Give me a rock, paper and scissors and I will move the world.  CCFestoon"},
	{"049d5e26d4f10222cd841a119e38bd8d2e0d1129728688449575d4ff42b842c1", "If the enemy is within range, then so are you."},
	{"0e116838e3cc1c1a14cd045397e29b4d087aa11b0853fc69ec82e90330d60949", "It's well we cannot hear the screams/That we create in others' dreams."},
	{"4f7d8eb5bcf11de2a56b971021a444aa4eafd6ecd0f307b5109e4e776cd0fe46", "You remind me of a TV show, but that's all right: I watch it anyway."},
	{"61c0cc4c4bd8406d5120b3fb4ebc31ce87667c162f29468b3c779675a85aebce", "C is as portable as Stonehedge!!"},
	{"1fb2eb3688093c4a3f80cd87a5547e2ce940a4f923243a79a2a1e242220693ac", "Even if I could be Shakespeare, I think I should still choose to be Faraday. - A. Huxley"},
	{"395585ce30617b62c80b93e8208ce866d4edc811a177fdb4b82d3911d8696423", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule"},
	{"4f9b189a13d030838269dce846b16a1ce9ce81fe63e65de2f636863336a98fe6", "How can you write a big system without C++?  -Paul Glick"},
}

func TestOpenPGP_DecodeCert(t *testing.T) {
	cases := []struct {
		cert string
		err  string
	}{
		{`-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAlRuRnThUjU8/prwYxbty
WPT9pURI3lbsKMiB6Fn/VHOKE13p4D8xgOCADpdRagdT6n4etr9atzDKUSvpMtR3
CP5noNc97WiNCggBjVWhs7szEe8ugyqF23XwpHQ6uV1LKH50m92MbOWfCtjU9p/x
qhNpQQ1AZhqNy5Gevap5k8XzRmjSldNAFZMY7Yv3Gi+nyCwGwpVtBUwhuLzgNFK/
yDtw2WcWmUU7NuC8Q6MWvPebxVtCfVp/iQU6q60yyt6aGOBkhAX0LpKAEhKidixY
nP9PNVBvxgu3XZ4P36gZV6+ummKdBVnc3NqwBLu5+CcdRdusmHPHd5pHf4/38Z3/
6qU2a/fPvWzceVTEgZ47QjFMTCTmCwNt29cvi7zZeQzjtwQgn4ipN9NibRH/Ax/q
TbIzHfrJ1xa2RteWSdFjwtxi9C20HUkjXSeI4YlzQMH0fPX6KCE7aVePTOnB69I/
a9/q96DiXZajwlpq3wFctrs1oXqBp5DVrCIj8hU2wNgB7LtQ1mCtsYz//heai0K9
PhE4X6hiE0YmeAZjR0uHl8M/5aW9xCoJ72+12kKpWAa0SFRWLy6FejNYCYpkupVJ
yecLk/4L1W0l6jQQZnWErXZYe0PNFcmwGXy1Rep83kfBRNKRy5tvocalLlwXLdUk
AIU+2GKjyT3iMuzZxxFxPFMCAwEAAQ==
-----END PUBLIC KEY-----`,
			"Invalid CERTIFICATE PEM format",
		},
		{
			`-----BEGIN CERTIFICATE-----
MIIE5DCCAsygAwIBAgIUYMHXek5TIDYT2Gv/bX1ApgbnmmswDQYJKoZIhvcNAQEN
BQAwgYAxCzAJBgNVBAYTAlVTMQswCQYDVQQHEwJDQTEcMBoGA1UEChMTU2FsZXNm
b3JjZSBJbnRlcm5hbDEPMA0GA1UECxMGcmFwaHR5MTUwMwYDVQQDDCxbVEVTVF0g
c2FsZXNmb3JjZS5jb20gSG9zdCBJbnRlZ3JpdHkgUm9vdCBDQTAeFw0xNzEyMzAw
MDMzMDBaFw0yMjEyMjkwMDMzMDBaMHUxCzAJBgNVBAYTAlVTMQswCQYDVQQHEwJD
QTEXMBUGA1UEChMOU2FsZXNmb3JjZS5jb20xEzARBgNVBAsTCnN0YW1weS1kZXYx
KzApBgNVBAMMIltURVNUXSBIb3N0IEludGVncml0eSBDb2RlIFNpZ25pbmcwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCrwJjC20xljx+hmF8/ZrbJ74is
X4jzU/O3MEdnbLlftwZxg/qjtIJLDtgQMKxfnyoiHVYxMuQgTqvKiJ1IxmYewq4f
QdmriF26st/QHD4xD+6b+n6K1X9iuj6sNfjd1vn0xnjBUXOAjm1e9X4mjWxFzIVh
ZDGqcQMs2uR8MbJBSkrfDpxTvFcvm5bkFf/uwgnMaR/wQylGjG8OzTZtrDM97oJo
/bZ3gmTRiepD8iOLnWzjir6e2QRmIzaqzCnqcxKG23WZIzA8JJzSdb/0hDhPrCG/
QzSMv4foLIv7m14WZb58eiqTdEa9vBLreQNRdOMH6D7YrYmTU2pW5bX9duJdAgMB
AAGjYDBeMA4GA1UdDwEB/wQEAwIFoDAMBgNVHRMBAf8EAjAAMB0GA1UdDgQWBBRn
M2rQ3rn8P8Xo+/4BlF247x4dDjAfBgNVHSMEGDAWgBT4YnyvE5eI3XQxsIMYgCkL
3d6bOTANBgkqhkiG9w0BAQ0FAAOCAgEAfcGIh/N3SRVfWxNDDO+hoRbn3fjt5JZ2
j69lgYxu9RZwCn6yzqZyTOdUPagsHjIt+vSHHqPaqY1KclsLp5qyajhg2suNIKk9
S4M8R0de69EyHhEzrOUTCEZGOn/qD+QOsEh1wotBPStrBwsPLowmatQuT9nMTS30
2syyJRMxcIovT42C96bsZDbZ9LCggmZxgNyR/4HsudwErJXEbKMrfgj19IKNAmwE
0aTsR8bwL1z6xPoAsj4qV4dUKgyataNeLHNM7heBp+mS/syrPh7Atbxc8zeTVIPt
HbBGJZhfyWhFFn1hLtNzIYXojY0JnQDCo+lhb4TjA+WhN3tUmjIhzoc+RfuXLDhs
9hx1QOiOK2tqhE9A5ktfNuTqYG5uNAYFkn/UNqKCYleNvU5oJTsWXlJjHUn6nZMw
7lcOmIoueJ8Yv6NsJyiIaEALH+OFHpBSP2XwT99aLxB+m9ZQga31gdOjnbFjZ7xO
NFonkE+JtP9G9ZBdblYCalVSh7HGQq4FJJ5tnYafETeTp/2uZjcMd153mu2XVg80
RGengvzTQrA1mh05A+a+gDsIC4hqvAZ5/m05wPb16OyGXuTiDidSLFk11aMQNMMS
q3wTa6b37uBAFVQEWRninTS5CD/rrmZ0j39cpLZtUKpMDluIqE0WAhqc5PrUr7vm
8RU1g4Qz0Lg=
-----END CERTIFICATE-----`,
			"",
		},
	}

	for i, tc := range cases {
		pk, err := ConvertTopX509CertificateToPGPPublicKey(tc.cert)
		if tc.err != "" {
			assert.Error(t, err, fmt.Sprintf("[%d] %s", i, tc.err))
		} else {
			assert.NoError(t, err, fmt.Sprintf("[%d] %s", i, tc.err))
			assert.NotNil(t, pk)
		}
	}
}

func TestOpenPGP_Hash256(t *testing.T) {
	// regular hash
	for i, tc := range golden256 {
		h, _, err := hashForSignature(crypto.SHA256, packet.SigTypeBinary)
		assert.NoError(t, err)

		h.Write([]byte(tc.in))

		val := hex.EncodeToString(h.Sum(nil))
		assert.Equal(t, tc.out, val, fmt.Sprintf("[%d] Hash %s", i, val))

		r := sha256.New()
		io.WriteString(r, tc.in[0:len(tc.in)/2])
		io.WriteString(r, tc.in[len(tc.in)/2:])

		val = hex.EncodeToString(r.Sum(nil))
		assert.Equal(t, tc.out, val, fmt.Sprintf("[%d] Resumable Hash %s", i, val))
	}
}

func Test_CreateSignaturePGP(t *testing.T) {
	pemCert, pemKey, err := testify.MakeSelfCertRSAPem(720)
	require.NoError(t, err)

	certs, err := certutil.ParseChainFromPEM(pemCert)
	require.NoError(t, err)

	pgpPublicKey, err := ConvertTopX509CertificateToPGPPublicKey(string(pemCert))
	require.NoError(t, err)

	pgpPrivateKey, err := ConvertPemToPgpPrivateKey(certs[0].NotBefore, []byte(pemKey))
	require.NoError(t, err)

	assert.Equal(t, pgpPublicKey.KeyId, pgpPrivateKey.KeyId)

	ctx, err := CreateOpenPGPEntity(pgpPublicKey, pgpPrivateKey, nil, OpenPGPEntitySignAll)
	require.NoError(t, err)

	var buf bytes.Buffer
	sigWriter := io.Writer(&buf)

	data := []byte{'T', 'e', 's', 't', ' ', 'd', 'a', 't', 'a', ' ', 't', 'o', ' ', 's', 'i', 'g', 'n'}
	message := bytes.NewReader(data)

	// Produce PGP signature
	err = OpenpgpDetachSign(message, sigWriter, ctx, OpenpgpSigTypeBinary, nil)
	require.NoError(t, err)

	signatureStr := buf.String()

	t.Logf(string(pemCert))
	t.Logf(signatureStr)

	signed := sha256.New()
	_, err = signed.Write(data)
	require.NoError(t, err)

	pubkey, err := ConvertTopX509CertificateToPGPPublicKey(string(pemCert))
	require.NoError(t, err)

	err = VerifySignaturePGP(signed, signatureStr, pubkey)
	require.NoError(t, err)

	pem, err := EncodePGPEntityToPEM(ctx)
	require.NoError(t, err)
	t.Logf(string(pem))

	kr, err := KeyRing(pem)
	require.NoError(t, err)
	assert.Empty(t, kr.DecryptionKeys())
	assert.NotEmpty(t, kr.KeysById(ctx.PrivateKey.KeyId))
}

func Test_VerifySignaturePGP(t *testing.T) {
	pemCert := `
-----BEGIN CERTIFICATE-----
MIIBITCBzKADAgECAghNZYIhB/z9UjANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQD
Ewlsb2NhbGhvc3QwHhcNMTgwNjExMjIzMjIyWhcNMTgwNzExMjMzMjIyWjAUMRIw
EAYDVQQDEwlsb2NhbGhvc3QwXDANBgkqhkiG9w0BAQEFAANLADBIAkEA2thU55uv
D4YJ2GQAia2QRzvtQRliGnCNgRxbPDrUiJrqQPVHgKhMQtcgzDzh4Qj8p17a7VOd
s+vQ4dBqHfX8DQIDAQABowIwADANBgkqhkiG9w0BAQsFAANBAMle0njI+8j08uMR
r2+c7UhMwuwNQIGa7Agp7ZFGTUdCT0ICDR11zMgYMTFwfanzM5SqEuX8bMtbyB5i
1VFh6Yc=
-----END CERTIFICATE-----
`
	signature := `
-----BEGIN PGP SIGNATURE-----

wlwEAAEIABAFAlsfBoYJEM+V0x+DrirOAADafwIAXwlrlYMOC94K/c17efcqrqHd
aWSp29Dwgi9rQ9rwWzCfNf6eJxjNZpOrgfpB3h1ZJD+VI2QnaM+m2z7jPqgRqA==
=9hGz
-----END PGP SIGNATURE-----
`

	data := []byte{'T', 'e', 's', 't', ' ', 'd', 'a', 't', 'a', ' ', 't', 'o', ' ', 's', 'i', 'g', 'n'}

	signed := sha256.New()
	_, err := signed.Write(data)
	require.NoError(t, err)

	pubkey, err := ConvertTopX509CertificateToPGPPublicKey(string(pemCert))
	require.NoError(t, err)

	err = VerifySignaturePGP(signed, signature, pubkey)
	require.NoError(t, err)
}

func Test_ConvertPublicKeyToPGP(t *testing.T) {
	now := time.Now().UTC()

	// ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	require.NoError(t, err)
	pgpPublicKey := ConvertPublicKeyToPGP(now, &privateKey.PublicKey)
	pgpPrivateKey := packet.NewECDSAPrivateKey(now, privateKey)
	assert.Equal(t, pgpPublicKey.KeyId, pgpPrivateKey.KeyId)
	algo, err := GetPgpPubkeyAlgo(pgpPublicKey)
	require.NoError(t, err)
	assert.Equal(t, "ECDSA", algo)

	// rsa key pair
	rsaPrivateKey, err := rsa.GenerateKey(crand.Reader, 1024)
	require.NoError(t, err)
	pgpPublicKey = ConvertPublicKeyToPGP(now, &rsaPrivateKey.PublicKey)
	pgpPrivateKey = packet.NewRSAPrivateKey(now, rsaPrivateKey)
	assert.Equal(t, pgpPublicKey.KeyId, pgpPrivateKey.KeyId)
	algo, err = GetPgpPubkeyAlgo(pgpPublicKey)
	require.NoError(t, err)
	assert.Equal(t, "RSA", algo)

	assert.Panics(t, func() {
		ConvertPublicKeyToPGP(now, struct{}{})
	})
}

func Test_ConvertLocalSignerToPgpPrivateKey_RSA(t *testing.T) {
	now := time.Now().UTC()
	crt, pvk, err := testify.MakeSelfCertRSA(720)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		pubkey := Convert509CertificateToPGPPublicKey(crt)
		require.NotNil(t, pubkey)
	})

	assert.NotPanics(t, func() {
		gpgpvk := ConvertLocalSignerToPgpPrivateKey(now, pvk.(*rsa.PrivateKey))
		require.NotNil(t, gpgpvk)
	})

	_, err = ConvertToPacketPrivateKey(now, pvk.(*rsa.PrivateKey))
	require.NoError(t, err)
}

func Test_ConvertLocalSignerToPgpPrivateKey_ECDSA(t *testing.T) {
	now := time.Now().UTC()
	crt, pvk, err := testify.MakeSelfCertECDSA(720)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		pubkey := Convert509CertificateToPGPPublicKey(crt)
		require.NotNil(t, pubkey)
	})

	assert.NotPanics(t, func() {
		gpgpvk := ConvertLocalSignerToPgpPrivateKey(now, pvk.(*ecdsa.PrivateKey))
		require.NotNil(t, gpgpvk)
	})

	_, err = ConvertToPacketPrivateKey(now, pvk.(*ecdsa.PrivateKey))
	require.NoError(t, err)
}

func Test_ConvertPemToPgpPrivateKey(t *testing.T) {
	now := time.Now().UTC()
	_, keyPem, err := testify.MakeSelfCertRSAPem(720)
	require.NoError(t, err)
	_, err = ConvertPemToPgpPrivateKey(now, keyPem)
	require.NoError(t, err)

	_, keyPem, err = testify.MakeSelfCertECDSAPem(720)
	require.NoError(t, err)
	_, err = ConvertPemToPgpPrivateKey(now, keyPem)
	require.NoError(t, err)

}

func Test_EncodePGPEntity(t *testing.T) {
	// rsa key pair
	rsaPrivateKey, err := rsa.GenerateKey(crand.Reader, 1024)
	require.NoError(t, err)

	now := time.Now().UTC()
	pgpPublicKey := ConvertPublicKeyToPGP(now, &rsaPrivateKey.PublicKey)
	pgpPrivateKey := packet.NewRSAPrivateKey(now, rsaPrivateKey)

	assert.Equal(t, pgpPublicKey.KeyId, pgpPrivateKey.KeyId)

	entity, err := CreateOpenPGPEntity(pgpPublicKey, pgpPrivateKey, nil, OpenPGPEntitySignAll)
	require.NoError(t, err)

	pem, err := EncodePGPEntityToPEM(entity)
	require.NoError(t, err)
	t.Logf(string(pem))

	entity2, err := DecodePGPEntityFromPEM(bytes.NewReader(pem))
	require.NoError(t, err)
	assert.Equal(t, entity.PrimaryKey.KeyId, entity2.PrimaryKey.KeyId)

	// Ensure that string produces the same
	entity2, err = DecodePGPEntityFromPEM(strings.NewReader(string(pem)))
	require.NoError(t, err)
	assert.Equal(t, entity.PrimaryKey.KeyId, entity2.PrimaryKey.KeyId)
}
