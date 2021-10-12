package authority

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"io"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/go-phorce/dolly/xpki/csr"
	"github.com/pkg/errors"
)

var (
	supportedKeyHash = []crypto.Hash{crypto.SHA1, crypto.SHA256, crypto.SHA384, crypto.SHA512}
)

// Issuer of certificates
type Issuer struct {
	cfg        IssuerConfig
	label      string
	skid       string // Subject Key ID
	signer     crypto.Signer
	sigAlgo    x509.SignatureAlgorithm
	bundle     *certutil.Bundle
	crlRenewal time.Duration
	crlExpiry  time.Duration
	ocspExpiry time.Duration
	crlURL     string
	aiaURL     string
	ocspURL    string

	// cabundlePEM contains PEM encoded certs for the issuer,
	// this bundle includes Issuing cert itself and its parents.
	cabundlePEM string

	keyHash  map[crypto.Hash][]byte
	nameHash map[crypto.Hash][]byte
}

// Bundle returns certificates bundle
func (ca *Issuer) Bundle() *certutil.Bundle {
	return ca.bundle
}

// PEM returns PEM encoded certs for the issuer
func (ca *Issuer) PEM() string {
	return ca.cabundlePEM
}

// CrlURL returns CRL DP URL
func (ca *Issuer) CrlURL() string {
	return ca.crlURL
}

// OcspURL returns OCSP URL
func (ca *Issuer) OcspURL() string {
	return ca.ocspURL
}

// AiaURL returns AIA URL
func (ca *Issuer) AiaURL() string {
	return ca.aiaURL
}

// Label returns label of the issuer
func (ca *Issuer) Label() string {
	return ca.label
}

// SubjectKID returns Subject Key ID
func (ca *Issuer) SubjectKID() string {
	return ca.skid
}

// Signer returns crypto.Signer
func (ca *Issuer) Signer() crypto.Signer {
	return ca.signer
}

// KeyHash returns key hash
func (ca *Issuer) KeyHash(h crypto.Hash) []byte {
	return ca.keyHash[h]
}

// CrlRenewal is duration for CRL renewal interval
func (ca *Issuer) CrlRenewal() time.Duration {
	return ca.crlRenewal
}

// CrlExpiry is duration for CRL next update interval
func (ca *Issuer) CrlExpiry() time.Duration {
	return ca.crlExpiry
}

// OcspExpiry is duration for OCSP next update interval
func (ca *Issuer) OcspExpiry() time.Duration {
	return ca.ocspExpiry
}

// Profile returns CertProfile
func (ca *Issuer) Profile(name string) *CertProfile {
	return ca.cfg.Profiles[name]
}

// NewIssuer creates Issuer from provided configuration
func NewIssuer(cfg *IssuerConfig, prov *cryptoprov.Crypto) (*Issuer, error) {
	// ensure that signer can be created before the key is generated
	cryptoSigner, err := NewSignerFromFromFile(
		prov,
		cfg.KeyFile)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to create signer")
	}

	// Build the bundle and register the CA cert
	var intCAbytes, rootBytes []byte
	if cfg.CABundleFile != "" {
		intCAbytes, err = ioutil.ReadFile(cfg.CABundleFile)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to load ca-bundle")
		}
	}

	if cfg.RootBundleFile != "" {
		rootBytes, err = ioutil.ReadFile(cfg.RootBundleFile)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to load root-bundle")
		}
	}

	certBytes, err := ioutil.ReadFile(cfg.CertFile)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load cert")
	}
	issuer, err := CreateIssuer(cfg, certBytes, intCAbytes, rootBytes, cryptoSigner)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return issuer, nil
}

// CreateIssuer returns Issuer created directly from crypto.Signer,
// this method is mostly used for testing
func CreateIssuer(cfg *IssuerConfig, certBytes, intCAbytes, rootBytes []byte, signer crypto.Signer) (*Issuer, error) {
	label := cfg.Label
	bundle, status, err := certutil.VerifyBundleFromPEM(certBytes, intCAbytes, rootBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create signing CA cert bundle")
	}
	if status.IsUntrusted() {
		return nil, errors.WithMessagef(err, "bundle is invalid: label=%s, cn=%q, expiresAt=%q, expiringSKU=[%v], untrusted=[%v]",
			label,
			bundle.Subject.CommonName,
			bundle.Expires.Format(time.RFC3339),
			strings.Join(status.ExpiringSKIs, ","),
			strings.Join(status.Untrusted, ","),
		)
	}

	var crlRenewal, crlExpiry, ocspExpiry time.Duration
	var crl, aia, ocsp string
	if cfg.AIA != nil {
		crl = strings.Replace(cfg.AIA.CrlURL, "${ISSUER_ID}", bundle.SubjectID, -1)
		aia = strings.Replace(cfg.AIA.AiaURL, "${ISSUER_ID}", bundle.SubjectID, -1)
		ocsp = strings.Replace(cfg.AIA.OcspURL, "${ISSUER_ID}", bundle.SubjectID, -1)
		crlRenewal = cfg.AIA.CRLRenewal
		crlExpiry = cfg.AIA.CRLExpiry
		ocspExpiry = cfg.AIA.OCSPExpiry
	}

	keyHash := make(map[crypto.Hash][]byte)
	nameHash := make(map[crypto.Hash][]byte)

	for _, h := range supportedKeyHash {
		// OCSP requires Hash of the Key without Tag:
		/// issuerKeyHash is the hash of the issuer's public key.  The hash
		// shall be calculated over the value (excluding tag and length) of
		// the subject public key field in the issuer's certificate.
		var publicKeyInfo struct {
			Algorithm pkix.AlgorithmIdentifier
			PublicKey asn1.BitString
		}
		_, err = asn1.Unmarshal(bundle.Cert.RawSubjectPublicKeyInfo, &publicKeyInfo)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to decode SubjectPublicKeyInfo")
		}

		keyHash[h] = certutil.Digest(h, publicKeyInfo.PublicKey.RightAlign())
		nameHash[h] = certutil.Digest(h, bundle.Cert.RawSubject)

		logger.Infof("label=%s, alg=%s, keyHash=%s, nameHash=%s",
			label, certutil.HashAlgoToStr(h), hex.EncodeToString(keyHash[h]), hex.EncodeToString(nameHash[h]))
	}

	cabundlePEM := strings.TrimSpace(bundle.CertPEM)
	if bundle.CACertsPEM != "" {
		cabundlePEM = cabundlePEM + "\n" + strings.TrimSpace(bundle.CACertsPEM)
	}

	return &Issuer{
		cfg:         *cfg,
		skid:        certutil.GetSubjectKeyID(bundle.Cert),
		signer:      signer,
		sigAlgo:     csr.DefaultSigAlgo(signer),
		bundle:      bundle,
		label:       label,
		crlURL:      crl,
		aiaURL:      aia,
		ocspURL:     ocsp,
		cabundlePEM: cabundlePEM,
		keyHash:     keyHash,
		nameHash:    nameHash,
		crlRenewal:  crlRenewal,
		crlExpiry:   crlExpiry,
		ocspExpiry:  ocspExpiry,
	}, nil
}

// Sign signs a new certificate based on the PEM-encoded
// certificate request with the specified profile.
func (ca *Issuer) Sign(req csr.SignRequest) (*x509.Certificate, []byte, error) {
	profileName := req.Profile
	if profileName == "" {
		profileName = "default"
	}
	profile := ca.cfg.Profiles[profileName]
	if profile == nil {
		return nil, nil, errors.New("unsupported profile: " + profileName)
	}

	csrTemplate, err := csr.ParsePEM([]byte(req.Request))
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	csrTemplate.SignatureAlgorithm = ca.sigAlgo

	// Copy out only the fields from the CSR authorized by policy.
	safeTemplate := x509.Certificate{}
	// If the profile contains no explicit whitelist, assume that all fields
	// should be copied from the CSR.
	if profile.AllowedCSRFields == nil {
		safeTemplate = *csrTemplate
	} else {
		if profile.AllowedCSRFields.Subject {
			safeTemplate.Subject = csrTemplate.Subject
		}
		if profile.AllowedCSRFields.DNSNames {
			safeTemplate.DNSNames = csrTemplate.DNSNames
		}
		if profile.AllowedCSRFields.IPAddresses {
			safeTemplate.IPAddresses = csrTemplate.IPAddresses
		}
		if profile.AllowedCSRFields.URIs {
			safeTemplate.URIs = csrTemplate.URIs
		}
		if profile.AllowedCSRFields.EmailAddresses {
			safeTemplate.EmailAddresses = csrTemplate.EmailAddresses
		}
		safeTemplate.PublicKeyAlgorithm = csrTemplate.PublicKeyAlgorithm
		safeTemplate.PublicKey = csrTemplate.PublicKey
		safeTemplate.SignatureAlgorithm = csrTemplate.SignatureAlgorithm
	}

	/*
		isSelfSign := ca.bundle == nil
			if safeTemplate.IsCA {
				if !profile.CAConstraint.IsCA {
					return nil, nil, errors.New("the policy disallows issuing CA certificate")
				}

				if !isSelfSign {
					caCert := ca.bundle.Cert
					if caCert.MaxPathLen > 0 {
						if safeTemplate.MaxPathLen >= caCert.MaxPathLen {
							return nil, nil, errors.New("the issuer disallows CA MaxPathLen extending")
						}
					} else if caCert.MaxPathLen == 0 && caCert.MaxPathLenZero {
						// signer has pathlen of 0, do not sign more intermediate CAs
						return nil, nil, errors.New("the issuer disallows issuing CA certificate")
					}
				}
			}
	*/

	csr.SetSAN(&safeTemplate, req.SAN)
	safeTemplate.Subject = csr.PopulateName(req.Subject, safeTemplate.Subject)

	// If there is a whitelist, ensure that both the Common Name, SAN DNSNames and Emails match
	if profile.AllowedNamesRegex != nil && safeTemplate.Subject.CommonName != "" {
		if !profile.AllowedNamesRegex.Match([]byte(safeTemplate.Subject.CommonName)) {
			return nil, nil, errors.New("CommonName does not match allowed list: " + safeTemplate.Subject.CommonName)
		}
	}
	if profile.AllowedDNSRegex != nil {
		for _, name := range safeTemplate.DNSNames {
			if !profile.AllowedDNSRegex.Match([]byte(name)) {
				return nil, nil, errors.New("DNS Name does not match allowed list: " + name)
			}
		}
	}
	if profile.AllowedEmailRegex != nil {
		for _, name := range safeTemplate.EmailAddresses {
			if !profile.AllowedEmailRegex.Match([]byte(name)) {
				return nil, nil, errors.New("Email does not match allowed list: " + name)
			}
		}
	}
	if profile.AllowedURIRegex != nil {
		for _, u := range safeTemplate.URIs {
			uri := u.String()
			if !profile.AllowedURIRegex.Match([]byte(uri)) {
				return nil, nil, errors.New("URI does not match allowed list: " + uri)
			}
		}
	}

	{
		// RFC 5280 4.1.2.2:
		// Certificate users MUST be able to handle serialNumber
		// values up to 20 octets.  Conforming CAs MUST NOT use
		// serialNumber values longer than 20 octets.
		serialNumber := make([]byte, 20)
		_, err = io.ReadFull(rand.Reader, serialNumber)
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "failed to generate serial number")
		}

		// SetBytes interprets buf as the bytes of a big-endian
		// unsigned integer. The leading byte should be masked
		// off to ensure it isn't negative.
		serialNumber[0] &= 0x7F

		safeTemplate.SerialNumber = new(big.Int).SetBytes(serialNumber)
	}

	if len(req.Extensions) > 0 {
		for _, ext := range req.Extensions {
			if !profile.IsAllowedExtention(ext.ID) {
				return nil, nil, errors.New("extension not allowed: " + ext.ID.String())
			}

			rawValue, err := hex.DecodeString(ext.Value)
			if err != nil {
				return nil, nil, errors.WithMessagef(err, "failed to decode extension")
			}

			safeTemplate.ExtraExtensions = append(safeTemplate.ExtraExtensions, pkix.Extension{
				Id:       asn1.ObjectIdentifier(ext.ID),
				Critical: ext.Critical,
				Value:    rawValue,
			})
		}
	}

	err = ca.fillTemplate(&safeTemplate, profile, req.NotBefore, req.NotAfter)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to populate template")
	}

	var certTBS = safeTemplate

	signedCertPEM, err := ca.sign(&certTBS)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	crt, err := certutil.ParseFromPEM(signedCertPEM)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	// TODO: register issued cert

	return crt, signedCertPEM, nil
}

func (ca *Issuer) sign(template *x509.Certificate) ([]byte, error) {
	var caCert *x509.Certificate

	if ca.bundle == nil {
		// self-signed
		if !template.IsCA {
			return nil, errors.New("CA template is not specified")
		}
		template.DNSNames = nil
		template.EmailAddresses = nil
		template.URIs = nil
		caCert = template
	} else {
		caCert = ca.bundle.Cert
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, template.PublicKey, ca.signer)
	if err != nil {
		return nil, errors.WithMessagef(err, "create certificate")
	}

	logger.Infof("serial=%d, CN=%q, URI=%v, DNS=%v, Email=%v",
		template.SerialNumber, template.Subject.CommonName, template.URIs, template.DNSNames, template.EmailAddresses)

	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	return cert, nil
}

func (ca *Issuer) fillTemplate(template *x509.Certificate, profile *CertProfile, notBefore, notAfter time.Time) error {
	ski, err := computeSKI(template)
	if err != nil {
		return err
	}

	/* for debugging
	js, _ := json.Marshal(profile)
	fmt.Printf("fillTemplate: %v, notBefore=%v, notAfter=%v", string(js),
		notBefore.Format(time.RFC3339),
		notAfter.Format(time.RFC3339))
	*/

	var (
		eku    []x509.ExtKeyUsage
		ku     x509.KeyUsage
		expiry time.Duration = profile.Expiry.TimeDuration()
	)

	if expiry == 0 && notAfter.IsZero() {
		return errors.Errorf("expiry is not set")
	}

	// The third value returned from Usages is a list of unknown key usages.
	// This should be used when validating the profile at load, and isn't used
	// here.
	ku, eku, _ = profile.Usages()
	if ku == 0 && len(eku) == 0 {
		return errors.Errorf("invalid profile: no key usages")
	}

	if notBefore.IsZero() {
		backdate := -1 * profile.Backdate.TimeDuration()
		if backdate == 0 {
			backdate = -5 * time.Minute
		}
		notBefore = time.Now().Round(time.Minute).Add(backdate)
	}
	if notAfter.IsZero() {
		notAfter = notBefore.Add(expiry)
	}
	// TODO: ensure that time from CSR does no exceed allowed in profile
	if template.NotBefore.IsZero() || template.NotBefore.Before(notBefore) {
		template.NotBefore = notBefore.UTC()
	}
	if template.NotAfter.IsZero() || notAfter.Before(template.NotAfter) {
		template.NotAfter = notAfter.UTC()
	}
	template.KeyUsage = ku
	template.ExtKeyUsage = eku
	template.BasicConstraintsValid = true
	template.IsCA = profile.CAConstraint.IsCA
	if template.IsCA {
		logger.Noticef("subject=%q, is_ca=true, MaxPathLen=%d", template.Subject.String(), profile.CAConstraint.MaxPathLen)
		template.MaxPathLen = profile.CAConstraint.MaxPathLen
		template.MaxPathLenZero = template.MaxPathLen == 0
		template.DNSNames = nil
		template.IPAddresses = nil
		template.EmailAddresses = nil
		template.URIs = nil
	}
	template.SubjectKeyId = ski

	// TODO: check if profile allows OCSP and CRL

	ocspURL := ca.OcspURL()
	if ocspURL != "" {
		template.OCSPServer = []string{ocspURL}
	}
	crlURL := ca.CrlURL()
	if crlURL != "" {
		template.CRLDistributionPoints = []string{crlURL}
	}
	issuerURL := ca.AiaURL()
	if issuerURL != "" {
		template.IssuingCertificateURL = []string{issuerURL}
	}
	if len(profile.Policies) != 0 {
		err = addPolicies(template, profile.Policies)
		if err != nil {
			return errors.WithMessagef(err, "invalid profile policies")
		}
	}
	if profile.OCSPNoCheck {
		ocspNoCheckExtension := pkix.Extension{
			Id:       asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 48, 1, 5},
			Critical: false,
			Value:    []byte{0x05, 0x00},
		}
		template.ExtraExtensions = append(template.ExtraExtensions, ocspNoCheckExtension)
	}

	return nil
}
