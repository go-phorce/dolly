package testca

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math"
	"math/big"
	"time"
)

type configuration struct {
	subject               *pkix.Name
	issuer                *Entity
	nextSN                *int64
	priv                  *crypto.Signer
	isCA                  bool
	notBefore             *time.Time
	notAfter              *time.Time
	keyUsage              x509.KeyUsage
	extKeyUsage           []x509.ExtKeyUsage
	extensions            []pkix.Extension
	crldpURL              []string
	issuingCertificateURL []string
	ocspServer            []string
	dnsNames              []string
}

func (c *configuration) generate() *Entity {
	templ := &x509.Certificate{
		Subject:               c.getSubject(),
		IsCA:                  c.isCA,
		BasicConstraintsValid: true,
		NotAfter:              c.getNotAfter(),
		NotBefore:             c.getNotBefore(),
		KeyUsage:              c.keyUsage,
		ExtKeyUsage:           c.extKeyUsage,
		Extensions:            c.extensions,
		IssuingCertificateURL: c.issuingCertificateURL,
		OCSPServer:            c.ocspServer,
		CRLDistributionPoints: c.crldpURL,
		DNSNames:              c.dnsNames,
	}

	var (
		parent   *x509.Certificate
		thisPriv = c.getPrivateKey()
		priv     crypto.Signer
	)

	if c.issuer != nil {
		parent = c.issuer.Certificate
		templ.SerialNumber = big.NewInt(c.issuer.IncrementSN())
		priv = c.issuer.PrivateKey
	} else {
		parent = templ
		templ.SerialNumber = randSN()
		priv = thisPriv
	}

	der, err := x509.CreateCertificate(rand.Reader, templ, parent, thisPriv.Public(), priv)
	if err != nil {
		panic(err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		panic(err)
	}

	return &Entity{
		Certificate: cert,
		PrivateKey:  thisPriv,
		Issuer:      c.issuer,
		NextSN:      c.getNextSN(),
	}
}

var (
	// DefaultCountry is the default subject Country.
	DefaultCountry = []string{"US"}

	// DefaultProvince is the default subject Province.
	DefaultProvince = []string{"CA"}

	// DefaultLocality is the default subject Locality.
	DefaultLocality = []string{"San Francisco"}

	// DefaultStreetAddress is the default subject StreetAddress.
	DefaultStreetAddress = []string(nil)

	// DefaultPostalCode is the default subject PostalCode.
	DefaultPostalCode = []string(nil)

	// DefaultCommonName is the default subject CommonName.
	DefaultCommonName = "[TEST]"

	cnCounter int64
)

func (c *configuration) getSubject() pkix.Name {
	if c.subject != nil {
		return *c.subject
	}

	var cn string
	if cnCounter == 0 {
		cn = DefaultCommonName
	} else {
		cn = fmt.Sprintf("%s #%d", DefaultCommonName, cnCounter)
	}
	cnCounter++

	return pkix.Name{
		Country:       DefaultCountry,
		Province:      DefaultProvince,
		Locality:      DefaultLocality,
		StreetAddress: DefaultStreetAddress,
		PostalCode:    DefaultPostalCode,
		CommonName:    cn,
	}
}

func (c *configuration) getNextSN() int64 {
	if c.nextSN == nil {
		sn := randSN().Int64()
		c.nextSN = &sn
	}

	return *c.nextSN
}

func randSN() *big.Int {
	i, err := rand.Int(rand.Reader, big.NewInt(int64(math.MaxInt64)))
	if err != nil {
		panic(err)
	}

	return i
}

func (c *configuration) getPrivateKey() crypto.Signer {
	if c.priv == nil {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}

		signer := crypto.Signer(priv)

		c.priv = &signer
	}

	return *c.priv
}

func (c *configuration) getNotBefore() time.Time {
	if c.notBefore == nil {
		return time.Unix(0, 0)
	}

	return *c.notBefore
}

func (c *configuration) getNotAfter() time.Time {
	if c.notAfter == nil {
		return time.Now().Add(time.Hour * 24 * 365 * 10)
	}

	return *c.notAfter
}

// Option is an option that can be passed to New().
type Option option
type option func(c *configuration)

// Subject is an Option that sets a entity's subject field.
func Subject(value pkix.Name) Option {
	return func(c *configuration) {
		c.subject = &value
	}
}

// NextSerialNumber is an Option that determines the SN of the next issued
// certificate.
func NextSerialNumber(value int64) Option {
	return func(c *configuration) {
		c.nextSN = &value
	}
}

// PrivateKey is an Option for setting the entity's private key.
func PrivateKey(value crypto.Signer) Option {
	return func(c *configuration) {
		c.priv = &value
	}
}

// Issuer is an Option for setting the entity's issuer.
func Issuer(value *Entity) Option {
	return func(c *configuration) {
		c.issuer = value
	}
}

// NotBefore is an Option for setting the entity's certificate's NotBefore.
func NotBefore(value time.Time) Option {
	return func(c *configuration) {
		c.notBefore = &value
	}
}

// NotAfter is an Option for setting the entity's certificate's NotAfter.
func NotAfter(value time.Time) Option {
	return func(c *configuration) {
		c.notAfter = &value
	}
}

// KeyUsage is an Option for setting the key usage.
func KeyUsage(value x509.KeyUsage) Option {
	return func(c *configuration) {
		c.keyUsage = value
	}
}

// ExtKeyUsage is an Option for setting the extended key usage.
func ExtKeyUsage(value x509.ExtKeyUsage) Option {
	return func(c *configuration) {
		c.extKeyUsage = append(c.extKeyUsage, value)
	}
}

// Extensions is an Option for setting extensions.
func Extensions(value []pkix.Extension) Option {
	return func(c *configuration) {
		c.extensions = value
	}
}

// IssuingCertificateURL is an Option for setting the entity's certificate's
// IssuingCertificateURL.
func IssuingCertificateURL(value ...string) Option {
	return func(c *configuration) {
		c.issuingCertificateURL = append(c.issuingCertificateURL, value...)
	}
}

// OCSPServer is an Option for setting the entity's certificate's OCSPServer.
func OCSPServer(value ...string) Option {
	return func(c *configuration) {
		c.ocspServer = append(c.ocspServer, value...)
	}
}

// CrlDpURL is an Option for setting the entity's certificate's CRL Distribution Point.
func CrlDpURL(value ...string) Option {
	return func(c *configuration) {
		c.crldpURL = append(c.crldpURL, value...)
	}
}

// DNSName is an Option for setting the SAN.
func DNSName(value ...string) Option {
	return func(c *configuration) {
		c.dnsNames = append(c.dnsNames, value...)
	}
}

// Authority is an Option for making an entity a certificate authority.
var Authority Option = func(c *configuration) {
	c.isCA = true
}
