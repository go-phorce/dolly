package csrprov

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/mail"

	"github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
	"github.com/pkg/errors"
)

// NewSigningCertificateRequest creates new request for signing certificate
func (c *Provider) NewSigningCertificateRequest(
	keyLabel, algo string, keySize int,
	CN string,
	names []X509Name,
	hosts []string,
) *CertificateRequest {
	return &CertificateRequest{
		KeyRequest: c.NewKeyRequest(keyLabel, algo, keySize, Signing),
		CN:         CN,
		Names:      names,
		Hosts:      hosts,
	}
}

// NewRoot creates a new root certificate from the certificate request.
func (c *Provider) NewRoot(req *CertificateRequest) (cert, csrPEM, key []byte, err error) {
	// RootCA fixup
	policy, err := MakeCAPolicy(req)
	if err != nil {
		err = errors.WithMessage(err, "ca policy failed")
		return
	}

	err = ValidateCSR(req)
	if err != nil {
		err = errors.WithMessage(err, "invalid request")
		return
	}

	csrPEM, gkey, keyID, err := c.ParseCsrRequest(req)
	if err != nil {
		key = nil
		err = errors.WithMessage(err, "process request")
		return
	}

	signkey := gkey.(crypto.Signer)

	uri, keyBytes, err := c.provider.ExportKey(keyID)
	if err != nil {
		err = errors.WithMessage(err, "key URI")
		return
	}

	if keyBytes == nil {
		key = []byte(uri)
	} else {
		key = keyBytes
	}

	s, err := local.NewSigner(signkey, nil, signer.DefaultSigAlgo(signkey), policy)
	if err != nil {
		err = errors.WithMessage(err, "create signer")
		return
	}

	signReq := signer.SignRequest{Hosts: req.Hosts, Request: string(csrPEM)}
	cert, err = s.Sign(signReq)

	return
}

// ProcessCsrRequest takes a certificate request and generates a key and
// CSR from it.
func (c *Provider) ProcessCsrRequest(req *CertificateRequest) (csrPEM, key []byte, keyID string, pub crypto.PublicKey, err error) {
	err = ValidateCSR(req)
	if err != nil {
		err = errors.WithMessage(err, "invalid request")
		return
	}

	var priv crypto.PrivateKey

	csrPEM, priv, keyID, err = c.ParseCsrRequest(req)
	if err != nil {
		key = nil
		err = errors.WithMessage(err, "process request")
		return
	}

	s, ok := priv.(crypto.Signer)
	if !ok {
		key = nil
		err = errors.WithMessage(err, "unable to convert key to crypto.Signer")
		return
	}
	pub = s.Public()

	uri, keyBytes, err := c.provider.ExportKey(keyID)
	if err != nil {
		err = errors.WithMessage(err, "key URI")
		return
	}

	if keyBytes == nil {
		key = []byte(uri)
	} else {
		key = keyBytes
	}

	return
}

// ParseCsrRequest takes a certificate request and generates a key and
// CSR from it. It does no validation -- caveat emptor. It will,
// however, fail if the key request is not valid (i.e., an unsupported
// curve or RSA key size). The lack of validation was specifically
// chosen to allow the end user to define a policy and validate the
// request appropriately before calling this function.
func (c *Provider) ParseCsrRequest(req *CertificateRequest) (csr []byte, priv crypto.PrivateKey, keyID string, err error) {
	if req.KeyRequest == nil {
		err = errors.New("invalid key request")
		return
	}

	logger.Infof("algo=%s, size=%d", req.KeyRequest.Algo(), req.KeyRequest.Size())

	priv, err = req.KeyRequest.Generate()
	if err != nil {
		err = errors.WithMessage(err, "generate key")
		return
	}

	var label string
	keyID, label, err = c.provider.IdentifyKey(priv)
	if err != nil {
		err = errors.WithMessage(err, "identify key")
		return
	}

	logger.Infof("generated key, id=%q, label=%q", keyID, label)

	var tpl = x509.CertificateRequest{
		Subject:            req.Name(),
		SignatureAlgorithm: req.KeyRequest.SigAlgo(),
	}

	for i := range req.Hosts {
		if ip := net.ParseIP(req.Hosts[i]); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		} else if email, err := mail.ParseAddress(req.Hosts[i]); err == nil && email != nil {
			tpl.EmailAddresses = append(tpl.EmailAddresses, req.Hosts[i])
		} else {
			tpl.DNSNames = append(tpl.DNSNames, req.Hosts[i])
		}
	}

	csr, err = x509.CreateCertificateRequest(rand.Reader, &tpl, priv)
	if err != nil {
		logger.Errorf("failed to generate a CSR: %v", errors.WithStack(err))
		err = errors.WithMessage(err, "generate csr")
		return
	}
	block := pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csr,
	}

	csr = pem.EncodeToMemory(&block)

	return
}
