package gpg

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"hash"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xpki", "gpg")

// OpenpgpSignatureType represents the different semantic meanings of an OpenPGP
// signature. See RFC 4880, section 5.2.1.
type OpenpgpSignatureType packet.SignatureType

const (
	// OpenpgpSigTypeBinary specifies Binary signature format
	OpenpgpSigTypeBinary OpenpgpSignatureType = 0
	// OpenpgpSigTypeText specifies Text signature format
	OpenpgpSigTypeText = 1
)

// ConvertTopX509CertificateToPGPPublicKey converts certificate in PEM fromat to PGP public key
func ConvertTopX509CertificateToPGPPublicKey(certificateChainPem string) (*packet.PublicKey, error) {
	block, _ := pem.Decode([]byte(certificateChainPem))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.Errorf("Invalid CERTIFICATE PEM format: %q", certificateChainPem)
	}

	var x509Certificate *x509.Certificate
	x509Certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Annotate(err, "failed to parse certificate")
	}

	return Convert509CertificateToPGPPublicKey(x509Certificate), nil
}

// Convert509CertificateToPGPPublicKey returns PGP public key from x509.Certificate
func Convert509CertificateToPGPPublicKey(c *x509.Certificate) *packet.PublicKey {
	return ConvertPublicKeyToPGP(c.NotBefore, c.PublicKey)
}

// ConvertPublicKeyToPGP returns PGP public key
func ConvertPublicKeyToPGP(creationTime time.Time, pub crypto.PublicKey) *packet.PublicKey {
	var pgpPublicKey *packet.PublicKey

	switch pub.(type) {
	case *rsa.PublicKey:
		rsaPublicKey := pub.(*rsa.PublicKey)
		pgpPublicKey = packet.NewRSAPublicKey(creationTime, rsaPublicKey)
	case *ecdsa.PublicKey:
		ecdsaPublicKey := pub.(*ecdsa.PublicKey)
		pgpPublicKey = packet.NewECDSAPublicKey(creationTime, ecdsaPublicKey)
	default:
		logger.Panicf("unknown type of public key: %s", reflect.TypeOf(pub))
	}

	return pgpPublicKey
}

// ConvertLocalSignerToPgpPrivateKey creates a sign-only PrivateKey from a crypto.Signer that
// implements RSA or ECDSA.
func ConvertLocalSignerToPgpPrivateKey(creationTime time.Time, signer crypto.Signer) *packet.PrivateKey {
	pk := new(packet.PrivateKey)
	switch pubkey := signer.Public().(type) {
	case *rsa.PublicKey:
		pk.PublicKey = *packet.NewRSAPublicKey(creationTime, pubkey)
		pk.PubKeyAlgo = packet.PubKeyAlgoRSA
	case rsa.PublicKey:
		pk.PublicKey = *packet.NewRSAPublicKey(creationTime, &pubkey)
		pk.PubKeyAlgo = packet.PubKeyAlgoRSA
	case *ecdsa.PublicKey:
		pk.PublicKey = *packet.NewECDSAPublicKey(creationTime, pubkey)
	case ecdsa.PublicKey:
		pk.PublicKey = *packet.NewECDSAPublicKey(creationTime, &pubkey)
	default:
		panic("openpgp: unknown crypto.Signer type in ConvertLocalSignerToPgpPrivateKey")
	}
	pk.PrivateKey = signer
	return pk
}

// OpenPGPEntityOp specifies operation to perform on Entity
type OpenPGPEntityOp int

const (
	// OpenPGPEntityOpNone specifies not to perform any operation
	OpenPGPEntityOpNone OpenPGPEntityOp = 0

	// OpenPGPEntitySignSelf specifies to sign self
	OpenPGPEntitySignSelf OpenPGPEntityOp = 1 << iota // 1 << 0 which is 00000001
	// OpenPGPEntitySignSubkeys specifies to sign subkeys
	OpenPGPEntitySignSubkeys
	// OpenPGPEntitySignIdentity specifies to sign Identity
	OpenPGPEntitySignIdentity

	// OpenPGPEntitySignAll specifies to sign Identity, subkeys, self
	OpenPGPEntitySignAll = OpenPGPEntitySignSubkeys | OpenPGPEntitySignSelf // | OpenPGPEntitySignIdentity
)

// CreateOpenPGPEntity creates PGP signer from private and public keys
func CreateOpenPGPEntity(pubKey *packet.PublicKey, privKey *packet.PrivateKey, uid *packet.UserId, ops OpenPGPEntityOp) (*openpgp.Entity, error) {
	bits, err := pubKey.BitLength()
	if err != nil {
		bits = 2048
	}

	config := packet.Config{
		DefaultHash:            crypto.SHA256,
		DefaultCipher:          packet.CipherAES256,
		DefaultCompressionAlgo: packet.CompressionZLIB,
		CompressionConfig: &packet.CompressionConfig{
			Level: 9,
		},
		RSABits: int(bits),
	}

	if uid == nil {
		uid = packet.NewUserId("", "", "")
	}

	entity := &openpgp.Entity{
		PrimaryKey: pubKey,
		PrivateKey: privKey,
		Identities: make(map[string]*openpgp.Identity),
	}
	isPrimaryID := false

	selfSig := &packet.Signature{
		CreationTime: pubKey.CreationTime,
		SigType:      packet.SigTypePositiveCert,
		PubKeyAlgo:   pubKey.PubKeyAlgo,
		Hash:         config.Hash(),
		IsPrimaryId:  &isPrimaryID,
		FlagsValid:   true,
		FlagSign:     true,
		FlagCertify:  true,
		IssuerKeyId:  &entity.PrimaryKey.KeyId,
	}

	selfIdentity := &openpgp.Identity{
		Name:          uid.Name,
		UserId:        uid,
		SelfSignature: selfSig,
	}

	entity.Identities[uid.Id] = selfIdentity

	/*
		keyLifetimeSecs := uint32(86400 * 365)
		entity.Subkeys = make([]openpgp.Subkey, 1)
		entity.Subkeys[0] = openpgp.Subkey{
			PublicKey:  pubKey,
			PrivateKey: privKey,
			Sig: &packet.Signature{
				CreationTime:              pubKey.CreationTime,
				SigType:                   packet.SigTypeSubkeyBinding,
				PubKeyAlgo:                pubKey.PubKeyAlgo,
				Hash:                      config.Hash(),
				PreferredHash:             []uint8{8}, // SHA-256
				FlagsValid:                true,
				FlagEncryptStorage:        true,
				FlagEncryptCommunications: true,
				IssuerKeyId:               &entity.PrimaryKey.KeyId,
				KeyLifetimeSecs:           &keyLifetimeSecs,
			},
		}
	*/
	if privKey != nil {
		if privKey.KeyId != entity.PrimaryKey.KeyId {
			logger.Errorf("api=Entity, reason=key_id, pubkey_id=%d, privkey_id=%d",
				pubKey.KeyId, privKey.KeyId)
		}

		if ops&OpenPGPEntitySignSelf == OpenPGPEntitySignSelf {
			err = selfSig.SignUserId(uid.Id, pubKey, privKey, &config)
			if err != nil {
				return nil, errors.Annotate(err, "SignIdentity")
			}
			//			selfIdentity.Signatures = append(selfIdentity.Signatures, selfSig)
		}
		/*
			if ops&OpenPGPEntitySignIdentity == OpenPGPEntitySignIdentity {
				err = entity.SignIdentity(uid.Id, entity, &config)
				if err != nil {
					return nil, errors.Annotate(err, "SignIdentity")
				}
			}
		*/
		if ops&OpenPGPEntitySignSubkeys == OpenPGPEntitySignSubkeys {
			for _, subkey := range entity.Subkeys {
				err = subkey.Sig.SignKey(subkey.PublicKey, privKey, &config)
				if err != nil {
					return nil, errors.Annotate(err, "SignIdentity")
				}
				//				selfIdentity.Signatures = append(selfIdentity.Signatures, subkey.Sig)
			}
		}
	}

	return entity, nil
}

// OpenpgpDetachSign creates detached signature on message
func OpenpgpDetachSign(message io.Reader, w io.Writer, signer *openpgp.Entity, sigType OpenpgpSignatureType, config *packet.Config) (err error) {
	switch packet.SignatureType(sigType) {
	case packet.SigTypeBinary:
		return openpgp.ArmoredDetachSign(w, signer, message, config)
	case packet.SigTypeText:
		return openpgp.ArmoredDetachSignText(w, signer, message, config)
	default:
		return errors.New("unsupported signature type")
	}
}

// GetPgpPubkeyAlgo returns algorithm in RSA2048 or ECDSA format
func GetPgpPubkeyAlgo(pubkey *packet.PublicKey) (string, error) {
	var algo string
	switch pubkey.PubKeyAlgo {
	case packet.PubKeyAlgoECDSA:
		algo = "ECDSA"
	case packet.PubKeyAlgoRSA, packet.PubKeyAlgoRSASignOnly:
		algo = "RSA"
	case packet.PubKeyAlgoDSA:
		algo = "DSA"
	default:
		return "", errors.Errorf("Invalid key algorithm for signature:'%v", pubkey.PubKeyAlgo)
	}

	return algo, nil
}

// DecodeArmoredPgpSignature decodes PGP signature
func DecodeArmoredPgpSignature(armored io.Reader) (*packet.Signature, error) {
	block, err := armor.Decode(armored)
	if err != nil {
		return nil, errors.Annotate(err, "decoding OpenPGP Armor")
	}

	if block.Type != openpgp.SignatureType {
		return nil, errors.Errorf("invalid signature file: '%v", block.Type)
	}

	reader := packet.NewReader(block.Body)
	pkt, err := reader.Next()
	if err != nil {
		return nil, errors.Annotate(err, "reading signature")
	}

	sig, ok := pkt.(*packet.Signature)
	if !ok {
		return nil, errors.Annotate(err, "invalid signature")
	}
	return sig, nil
}

// hashForSignature returns a pair of hashes that can be used to verify a
// signature. The signature may specify that the contents of the signed message
// should be preprocessed (i.e. to normalize line endings). Thus this function
// returns two hashes. The second should be used to hash the message itself and
// performs any needed preprocessing.
func hashForSignature(hashID crypto.Hash, sigType packet.SignatureType) (hash.Hash, hash.Hash, error) {
	if !hashID.Available() {
		return nil, nil, errors.Errorf("hash not available: " + strconv.Itoa(int(hashID)))
	}
	h := hashID.New()

	switch sigType {
	case packet.SigTypeBinary:
		return h, h, nil
	case packet.SigTypeText:
		return h, openpgp.NewCanonicalTextHash(h), nil
	}

	return nil, nil, errors.Errorf("unsupported signature type: " + strconv.Itoa(int(sigType)))
}

// ConvertPemToPgpPrivateKey parses a PEM encoded private key.
func ConvertPemToPgpPrivateKey(creationTime time.Time, privateKeyPem []byte) (*packet.PrivateKey, error) {
	var pgpPrivateKey *packet.PrivateKey

	block, _ := pem.Decode(privateKeyPem)
	if block == nil {
		return nil, errors.Errorf("Invalid PRIVATE KEY PEM format: %q", privateKeyPem)
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		// TODO: Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
		// PKCS#1 private keys by default, while OpenSSL 1.0.0 generates PKCS#8 keys.
		// OpenSSL ecparam generates SEC1 EC private keys for ECDSA.

		rsaPrivateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, errors.Annotate(err, "failed to parse RSA private key")
		}

		pgpPrivateKey = packet.NewRSAPrivateKey(creationTime, rsaPrivateKey)
	case "EC PRIVATE KEY", "ECDSA PRIVATE KEY":
		ecPrivateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, errors.Annotate(err, "failed to parse EC private key")
		}

		pgpPrivateKey = packet.NewECDSAPrivateKey(creationTime, ecPrivateKey)
	default:
		return nil, errors.Errorf("unsupported PRIVATE KEY: %q", block.Type)
	}

	return pgpPrivateKey, nil
}

// VerifySignaturePGP verifies the signatures
func VerifySignaturePGP(signed hash.Hash, pemSignature string, pubkey *packet.PublicKey) error {
	sig, err := DecodeArmoredPgpSignature(strings.NewReader(pemSignature))
	if err != nil {
		return errors.Annotate(err, "decode armored PGP signature")
	}

	if sig.PubKeyAlgo != pubkey.PubKeyAlgo {
		logger.Infof("api=VerifySignaturePGP, reason=PubKeyAlgo, pubkey_alg=%v, sig_alg=%v",
			pubkey.PubKeyAlgo, sig.PubKeyAlgo)
		// Ensure the algs match
		if pubkey.PubKeyAlgo == packet.PubKeyAlgoRSASignOnly {
			pubkey.PubKeyAlgo = packet.PubKeyAlgoRSA
		}
		if sig.PubKeyAlgo == packet.PubKeyAlgoRSASignOnly {
			sig.PubKeyAlgo = packet.PubKeyAlgoRSA
		}
	}

	err = pubkey.VerifySignature(signed, sig)
	if err != nil {
		return errors.Annotate(err, "detached PGP signature")
	}

	return nil
}

// EncodePGPEntityToPEM returns PEM encoded Entity's Public Key
func EncodePGPEntityToPEM(e *openpgp.Entity) ([]byte, error) {
	comments := fmt.Sprintf(`# Key ID: %d (0x%x)
# Created: %s
# Fingerprint: %x
# Identities:
`, e.PrimaryKey.KeyId, e.PrimaryKey.KeyId, e.PrimaryKey.CreationTime.Format(time.RFC3339), e.PrimaryKey.Fingerprint)

	for _, ident := range e.Identities {
		comments = comments + fmt.Sprintf("#    %s\n", ident.UserId.Id)
	}

	b := bytes.NewBufferString(comments)
	w, err := armor.Encode(b, openpgp.PublicKeyType, make(map[string]string))
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = e.Serialize(w)
	if err != nil {
		return nil, errors.Trace(err)
	}
	w.Close()
	return b.Bytes(), nil
}

// DecodePGPEntityFromPEM reads Entity from the given io.Reader
func DecodePGPEntityFromPEM(r io.Reader) (*openpgp.Entity, error) {
	// decode PEM
	p, err := armor.Decode(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	packets := packet.NewReader(p.Body)
	e, err := openpgp.ReadEntity(packets)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return e, nil
}

// ConvertToPacketPrivateKey converts a private key interface to PKCS11PrivateKey type
func ConvertToPacketPrivateKey(creationTime time.Time, s crypto.Signer) (*packet.PrivateKey, error) {
	var pgpPubKey packet.PublicKey
	switch s.Public().(type) {
	case *rsa.PublicKey:
		pgpPubKey = *packet.NewRSAPublicKey(creationTime, s.Public().(*rsa.PublicKey))
		break
	case *ecdsa.PublicKey:
		pgpPubKey = *packet.NewECDSAPublicKey(creationTime, s.Public().(*ecdsa.PublicKey))
		break
	default:
		return nil, errors.New("internal error. Publickey is unknown")
	}

	priv := &packet.PrivateKey{
		PrivateKey: s,
		Encrypted:  false,
		PublicKey:  pgpPubKey,
	}

	return priv, nil
}
