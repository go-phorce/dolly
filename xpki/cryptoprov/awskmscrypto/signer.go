package awskmscrypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"io"
	"reflect"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/pkg/errors"
)

// Supported signature types by AWS KMS
const (
	SignRsaPssSha256   = "RSASSA_PSS_SHA_256"
	SignRsaPssSha384   = "RSASSA_PSS_SHA_384"
	SignRsaPssSha512   = "RSASSA_PSS_SHA_512"
	SignRsaPkcs1Sha256 = "RSASSA_PKCS1_V1_5_SHA_256"
	SignRsaPkcs1Sha384 = "RSASSA_PKCS1_V1_5_SHA_384"
	SignRsaPkcs1Sha512 = "RSASSA_PKCS1_V1_5_SHA_512"
)

// Signer implements crypto.Signer interface
type Signer struct {
	keyID string
	arn   string
	label string
	//signAlgo x509.SignatureAlgorithm
	signingAlgorithms []string
	pubKey            crypto.PublicKey
	kmsClient         KmsClient
}

// NewSigner creates new signer
func NewSigner(keyID string, label string, signingAlgorithms []string, publicKey crypto.PublicKey, kmsClient KmsClient) crypto.Signer {
	logger.Debugf("id=%s, label=%q, algos=%v", keyID, label, signingAlgorithms)
	return &Signer{
		keyID:             keyID,
		label:             label,
		signingAlgorithms: signingAlgorithms,
		pubKey:            publicKey,
		kmsClient:         kmsClient,
	}
}

// KeyID returns key id of the signer
func (s *Signer) KeyID() string {
	return s.keyID
}

// Label returns key label of the signer
func (s *Signer) Label() string {
	return s.label
}

// Public returns public key for the signer
func (s *Signer) Public() crypto.PublicKey {
	return s.pubKey
}

func (s *Signer) String() string {
	return fmt.Sprintf("id=%s, label=%s",
		s.KeyID(),
		s.Label(),
	)
}

// Sign implements signing operation
func (s *Signer) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	typ := "DIGEST"

	sigAlgo, err := sigAlgo(s.pubKey, opts)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to determine signature algorithm")
	}

	req := &kms.SignInput{
		KeyId:            &s.keyID,
		Message:          digest,
		MessageType:      &typ,
		SigningAlgorithm: &sigAlgo,
	}
	resp, err := s.kmsClient.Sign(req)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to sign")
	}
	return resp.Signature, nil
}

func sigAlgo(publicKey crypto.PublicKey, opts crypto.SignerOpts) (string, error) {
	var pubalgo string
	var pad string

	switch publicKey.(type) {
	case *rsa.PublicKey:
		pubalgo = "RSASSA_"

		switch t := opts.(type) {
		case *rsa.PSSOptions:
			pad = "PSS_"
			opts = t.Hash
		default:
			pad = "PKCS1_V1_5_"
		}
	case *ecdsa.PublicKey:
		pubalgo = "ECDSA_"
	default:
		return "", errors.Errorf("unknown type of public key: %s", reflect.TypeOf(publicKey))
	}

	var algo string
	switch opts.HashFunc() {
	case crypto.SHA256:
		algo = pubalgo + pad + "SHA_256"
	case crypto.SHA384:
		algo = pubalgo + pad + "SHA_384"
	case crypto.SHA512:
		algo = pubalgo + pad + "SHA_512"
	default:
		return "", errors.Errorf("unsupported hash: %s", reflect.TypeOf(opts))

	}
	return algo, nil
}
