package authority_test

import (
	"testing"
	"time"

	"github.com/go-phorce/dolly/xpki/authority"
	"github.com/go-phorce/dolly/xpki/csr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projFolder = "../"

func TestDefaultCertProfile(t *testing.T) {
	def := authority.DefaultCertProfile().Copy()
	require.NotNil(t, def)

	def.AllowedExtensions = []csr.OID{{1, 2, 3, 4, 5, 6, 8}}
	assert.Equal(t, time.Duration(10*time.Minute), def.Backdate.TimeDuration())
	assert.Equal(t, time.Duration(8760*time.Hour), def.Expiry.TimeDuration())
	assert.Equal(t, "default profile with Server and Client auth", def.Description)
	require.NotEmpty(t, def.Usage)
	assert.Contains(t, def.Usage, "signing")
	assert.Contains(t, def.Usage, "key encipherment")
	assert.Contains(t, def.Usage, "server auth")
	assert.Contains(t, def.Usage, "client auth")
	assert.NoError(t, def.Validate())
	assert.False(t, def.IsAllowedExtention(csr.OID{1, 2, 3, 4, 5, 6, 7}))
	assert.True(t, def.IsAllowedExtention(csr.OID{1, 2, 3, 4, 5, 6, 8}))
	assert.NotEmpty(t, def.AllowedExtensionsStrings())
}

func TestLoadInvalidConfigFile(t *testing.T) {
	tcases := []struct {
		file string
		err  string
	}{
		{"", "invalid path"},
		{"testdata/no_such_file", "unable to read configuration file: open testdata/no_such_file: no such file or directory"},
		{"testdata/invalid_default.json", "failed to unmarshal configuration: time: invalid duration \"invalid_expiry\""},
		{"testdata/invalid_empty.json", "no \"profiles\" configuration present"},
		{"testdata/invalid_server.json", "invalid configuration: invalid server profile: unknown usage: encipherment"},
		{"testdata/invalid_noexpiry.json", "invalid configuration: invalid noexpiry_profile profile: no expiry set"},
		{"testdata/invalid_nousage.json", "invalid configuration: invalid no_usage_profile profile: no usages specified"},
		{"testdata/invalid_allowedname.json", "invalid configuration: invalid withregex profile: failed to compile AllowedNames: error parsing regexp: missing closing ]: `[}`"},
		{"testdata/invalid_dns.json", "invalid configuration: invalid withregex profile: failed to compile AllowedDNS: error parsing regexp: missing closing ]: `[}`"},
		{"testdata/invalid_uri.json", "invalid configuration: invalid withregex profile: failed to compile AllowedURI: error parsing regexp: missing closing ]: `[}`"},
		{"testdata/invalid_email.json", "invalid configuration: invalid withregex profile: failed to compile AllowedEmail: error parsing regexp: missing closing ]: `[}`"},
		{"testdata/invalid_qualifier.json", "invalid configuration: invalid with-qt profile: invalid policy qualifier type: qt-type"},
	}
	for _, tc := range tcases {
		t.Run(tc.file, func(t *testing.T) {
			_, err := authority.LoadConfig(tc.file)
			require.Error(t, err)
			assert.Equal(t, tc.err, err.Error())
		})
	}

}

func TestLoadConfig(t *testing.T) {
	_, err := authority.LoadConfig("")
	require.Error(t, err)
	assert.Equal(t, "invalid path", err.Error())

	_, err = authority.LoadConfig("not_found")
	require.Error(t, err)
	assert.Equal(t, "unable to read configuration file: open not_found: no such file or directory", err.Error())

	cfg, err := authority.LoadConfig("testdata/ca-config.dev.json")
	require.NoError(t, err)
	require.NotEmpty(t, cfg.Profiles)

	cfg2 := cfg.Copy()
	assert.Equal(t, cfg, cfg2)

	def := cfg.DefaultCertProfile()
	require.NotNil(t, def)
	assert.Equal(t, time.Duration(30*time.Minute), def.Backdate.TimeDuration())
	assert.Equal(t, time.Duration(168*time.Hour), def.Expiry.TimeDuration())

	files := []string{
		"testdata/ca-config.dev.json",
		"testdata/ca-config.bootstrap.json",
		"testdata/ca-config.dev.yaml",
		"testdata/ca-config.bootstrap.yaml",
	}
	for _, path := range files {
		cfg, err := authority.LoadConfig(path)
		require.NoError(t, err, "failed to parse: %s", path)
		require.NotEmpty(t, cfg.Profiles)
	}
}

func TestCertProfile(t *testing.T) {
	p := authority.CertProfile{
		Expiry:       csr.OneYear,
		Usage:        []string{"signing", "any"},
		AllowedNames: "trusty*",
		AllowedDNS:   "^(www\\.)?trusty\\.com$",
		AllowedEmail: "^ca@trusty\\.com$",
		AllowedURI:   "^spifee://trysty/.*$",
		AllowedExtensions: []csr.OID{
			{1, 1000, 1, 1},
			{1, 1000, 1, 3},
		},
	}
	assert.NoError(t, p.Validate())
	assert.True(t, p.IsAllowedExtention(csr.OID{1, 1000, 1, 3}))
	assert.False(t, p.IsAllowedExtention(csr.OID{1, 1000, 1, 3, 1}))
}

func TestDefaultAuthority(t *testing.T) {
	a := &authority.CAConfig{}
	assert.Equal(t, authority.DefaultCRLExpiry, a.DefaultAIA.GetCRLExpiry())
	assert.Equal(t, authority.DefaultOCSPExpiry, a.DefaultAIA.GetOCSPExpiry())
	assert.Equal(t, authority.DefaultCRLRenewal, a.DefaultAIA.GetCRLRenewal())

	d := 1 * time.Hour
	a = &authority.CAConfig{
		DefaultAIA: &authority.AIAConfig{
			CRLExpiry:  d,
			OCSPExpiry: d,
			CRLRenewal: d,
		},
	}
	assert.Equal(t, time.Duration(d), a.DefaultAIA.GetCRLExpiry())
	assert.Equal(t, time.Duration(d), a.DefaultAIA.GetOCSPExpiry())
	assert.Equal(t, time.Duration(d), a.DefaultAIA.GetCRLRenewal())
}

func TestProfilePolicyIsAllowed(t *testing.T) {
	emptyPolicy := &authority.CertProfile{}
	policy1 := &authority.CertProfile{
		IssuerLabel:  "issuer1",
		AllowedRoles: []string{"allowed1"},
		DeniedRoles:  []string{"denied1"},
	}
	policy2 := &authority.CertProfile{
		IssuerLabel:  "issuer2",
		AllowedRoles: []string{"*"},
		DeniedRoles:  []string{"denied1"},
	}
	policy3 := &authority.CertProfile{
		IssuerLabel:  "issuer3",
		AllowedRoles: []string{"*"},
		DeniedRoles:  []string{"*"},
	}

	tcases := []struct {
		policy  *authority.CertProfile
		role    string
		allowed bool
	}{
		{
			policy:  emptyPolicy,
			role:    "roles1",
			allowed: true,
		},
		{
			policy:  emptyPolicy,
			role:    "",
			allowed: true,
		},
		{
			policy:  policy1,
			role:    "allowed1",
			allowed: true,
		},
		{
			policy:  policy1,
			role:    "denied1",
			allowed: false,
		},
		{
			policy:  policy2,
			role:    "any",
			allowed: true,
		},
		{
			policy:  policy3,
			role:    "any",
			allowed: false,
		},
	}

	for _, tc := range tcases {
		assert.Equal(t, tc.allowed, tc.policy.IsAllowed(tc.role), "[%s] %s: Allowed->%v, Denied->%v",
			tc.policy.IssuerLabel, tc.role, tc.policy.AllowedRoles, tc.policy.DeniedRoles)
	}
}
