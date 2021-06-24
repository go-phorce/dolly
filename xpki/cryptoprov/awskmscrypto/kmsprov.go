package awskmscrypto

import (
	"crypto"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xpki", "awskmscrypto")

// ProviderName specifies a provider name
const ProviderName = "AWSKMS"

// KmsClient interface
type KmsClient interface {
	CreateKey(input *kms.CreateKeyInput) (*kms.CreateKeyOutput, error)
	//IdentifyKey(priv crypto.PrivateKey) (keyID, label string, err error)
	ListKeys(options *kms.ListKeysInput) (*kms.ListKeysOutput, error)
	ScheduleKeyDeletion(input *kms.ScheduleKeyDeletionInput) (*kms.ScheduleKeyDeletionOutput, error)
	DescribeKey(input *kms.DescribeKeyInput) (*kms.DescribeKeyOutput, error)
	GetPublicKey(input *kms.GetPublicKeyInput) (*kms.GetPublicKeyOutput, error)
	Sign(input *kms.SignInput) (*kms.SignOutput, error)
}

// KmsClientFactory override for unittest
var KmsClientFactory = func(p client.ConfigProvider, cfgs ...*aws.Config) (KmsClient, error) {
	return kms.New(p, cfgs...), nil
}

// Provider implements Provider interface for KMS
type Provider struct {
	tc        cryptoprov.TokenConfig
	kmsClient KmsClient
	endpoint  string
	region    string
}

// Init configures Kms based hsm impl
func Init(tc cryptoprov.TokenConfig) (*Provider, error) {
	kmsAttributes := parseKmsAttributes(tc.Attributes())
	endpoint := kmsAttributes["Endpoint"]
	region := kmsAttributes["Region"]

	p := &Provider{
		endpoint: endpoint,
		region:   region,
		tc:       tc,
	}

	mySession := session.Must(session.NewSession())
	cfg := aws.NewConfig()
	if endpoint != "" {
		cfg = cfg.WithEndpoint(endpoint)
	}
	if region != "" {
		cfg = cfg.WithRegion(region)
	}

	var err error
	p.kmsClient, err = KmsClientFactory(mySession, cfg)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to create KMS client")
	}

	return p, nil
}

func parseKmsAttributes(attributes string) map[string]string {
	var kmsAttributes = make(map[string]string)

	attrs := strings.Split(attributes, ",")
	for _, v := range attrs {
		kmsAttr := strings.Split(v, "=")
		kmsAttributes[strings.TrimSpace(kmsAttr[0])] = strings.TrimSpace(kmsAttr[1])
	}

	return kmsAttributes
}

// Manufacturer returns manufacturer for the provider
func (p *Provider) Manufacturer() string {
	return p.tc.Manufacturer()
}

// Model returns model for the provider
func (p *Provider) Model() string {
	return p.tc.Model()
}

// CurrentSlotID returns current slot id. For KMS only one slot is assumed to be available.
func (p *Provider) CurrentSlotID() uint {
	return 0
}

// GenerateRSAKey creates signer using randomly generated RSA key
func (p *Provider) GenerateRSAKey(label string, bits int, purpose int) (crypto.PrivateKey, error) {
	usage := "SIGN_VERIFY"
	if purpose == 2 {
		usage = "ENCRYPT_DECRYPT"
	}

	specuKeyPairSpec := fmt.Sprintf("RSA_%d", bits)

	// 1. Create key in KMS
	input := &kms.CreateKeyInput{
		CustomerMasterKeySpec: &specuKeyPairSpec,
		KeyUsage:              &usage,
		Description:           &label,
	}
	resp, err := p.kmsClient.CreateKey(input)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to create key with label: %q", label)
	}

	keyID := aws.StringValue(resp.KeyMetadata.KeyId)
	arn := aws.StringValue(resp.KeyMetadata.Arn)

	logger.Infof("arn=%q, id=%q, label=%q",
		arn,
		keyID,
		label,
	)

	// 2. Retrieve public key from KMS
	pubKeyResp, err := p.kmsClient.GetPublicKey(&kms.GetPublicKeyInput{KeyId: &keyID})
	if err != nil {
		return nil, errors.Annotatef(err, "failed to get public key, id=%s", keyID)
	}

	pub, err := x509.ParsePKIXPublicKey(pubKeyResp.PublicKey)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to parse public key, id=%s", keyID)
	}
	signer := NewSigner(keyID, label, aws.StringValueSlice(resp.KeyMetadata.SigningAlgorithms), pub, p.kmsClient)

	return signer, nil
}

// GenerateECDSAKey creates signer using randomly generated ECDSA key
func (p *Provider) GenerateECDSAKey(label string, curve elliptic.Curve) (crypto.PrivateKey, error) {
	usage := "SIGN_VERIFY"

	var spec string
	switch curve {
	case elliptic.P256():
		spec = "ECC_NIST_P256"
	case elliptic.P384():
		spec = "ECC_NIST_P384"
	case elliptic.P521():
		spec = "ECC_NIST_P521"
	default:
		return nil, errors.New("unsupported curve")
	}

	// 1. Create key in KMS
	input := &kms.CreateKeyInput{
		CustomerMasterKeySpec: &spec,
		KeyUsage:              &usage,
		Description:           &label,
	}
	resp, err := p.kmsClient.CreateKey(input)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to create key with label: %q", label)
	}

	keyID := aws.StringValue(resp.KeyMetadata.KeyId)
	arn := aws.StringValue(resp.KeyMetadata.Arn)

	logger.Infof("arn=%q, id=%q, label=%q",
		arn,
		keyID,
		label,
	)

	// 2. Retrieve public key from KMS
	pubKeyResp, err := p.kmsClient.GetPublicKey(&kms.GetPublicKeyInput{KeyId: &keyID})
	if err != nil {
		return nil, errors.Annotatef(err, "failed to get public key, id=%s", keyID)
	}

	pub, err := x509.ParsePKIXPublicKey(pubKeyResp.PublicKey)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to parse public key, id=%s", keyID)
	}
	signer := NewSigner(keyID, label, aws.StringValueSlice(resp.KeyMetadata.SigningAlgorithms), pub, p.kmsClient)

	return signer, nil
}

// IdentifyKey returns key id and label for the given private key
func (p *Provider) IdentifyKey(priv crypto.PrivateKey) (keyID, label string, err error) {
	if s, ok := priv.(*Signer); ok {
		return s.KeyID(), s.Label(), nil
	}
	return "", "", errors.New("not supported key")
}

// GetKey returns pkcs11 uri for the given key id
func (p *Provider) GetKey(keyID string) (crypto.PrivateKey, error) {
	logger.Infof("api=GetKey, keyID=%s", keyID)

	ki, err := p.kmsClient.DescribeKey(&kms.DescribeKeyInput{KeyId: &keyID})
	if err != nil {
		return nil, errors.Annotatef(err, "failed to describe key, id=%s", keyID)
	}

	resp, err := p.kmsClient.GetPublicKey(&kms.GetPublicKeyInput{KeyId: &keyID})
	if err != nil {
		return nil, errors.Annotatef(err, "failed to get public key, id=%s", keyID)
	}

	pub, err := x509.ParsePKIXPublicKey(resp.PublicKey)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to parse public key, id=%s", keyID)
	}
	signer := NewSigner(keyID, aws.StringValue(ki.KeyMetadata.Description), aws.StringValueSlice(resp.SigningAlgorithms), pub, p.kmsClient)
	return signer, nil
}

// EnumTokens lists tokens. For KMS currentSlotOnly is ignored and only one slot is assumed to be available.
func (p *Provider) EnumTokens(currentSlotOnly bool, slotInfoFunc func(slotID uint, description, label, manufacturer, model, serial string) error) error {
	return slotInfoFunc(p.CurrentSlotID(),
		"",
		"",
		p.Manufacturer(),
		p.Model(),
		"")
}

// EnumKeys returns list of keys on the slot. For KMS slotID is ignored.
func (p *Provider) EnumKeys(slotID uint, prefix string, keyInfoFunc func(id, label, typ, class, currentVersionID string, creationTime *time.Time) error) error {
	logger.Tracef("api=EnumKeys, host=%s, slotID=%d, prefix=%q", p.endpoint, slotID, prefix)

	opts := &kms.ListKeysInput{}

	resp, err := p.kmsClient.ListKeys(opts)
	if err != nil {
		return errors.Trace(err)
	}

	keys := resp.Keys
	for _, k := range keys {
		ki, err := p.kmsClient.DescribeKey(&kms.DescribeKeyInput{KeyId: k.KeyId})
		if err != nil {
			return errors.Annotatef(err, "failed to describe key, id=%s", *k.KeyId)
		}
		if aws.StringValue(ki.KeyMetadata.KeyState) == "PendingDeletion" {
			continue
		}

		err = keyInfoFunc(
			aws.StringValue(k.KeyId),
			aws.StringValue(ki.KeyMetadata.Description),
			aws.StringValue(ki.KeyMetadata.KeyUsage),
			aws.StringValue(ki.KeyMetadata.Origin),
			aws.StringValue(ki.KeyMetadata.KeyState),
			ki.KeyMetadata.CreationDate,
		)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// DestroyKeyPairOnSlot destroys key pair on slot. For KMS slotID is ignored and KMS retire API is used to destroy the key.
func (p *Provider) DestroyKeyPairOnSlot(slotID uint, keyID string) error {
	resp, err := p.kmsClient.ScheduleKeyDeletion(&kms.ScheduleKeyDeletionInput{
		KeyId: &keyID,
	})
	if err != nil {
		return errors.Annotatef(err, "failed to schedule key deletion: %s", keyID)
	}
	logger.Noticef("id=%s, deletion_time=%v",
		keyID, aws.TimeValue(resp.DeletionDate).Format(time.RFC3339))

	return nil
}

// KeyInfo retrieves info about key with the specified id
func (p *Provider) KeyInfo(slotID uint, keyID string, includePublic bool, keyInfoFunc func(id, label, typ, class, currentVersionID, pubKey string, creationTime *time.Time) error) error {
	resp, err := p.kmsClient.DescribeKey(&kms.DescribeKeyInput{KeyId: &keyID})
	if err != nil {
		return errors.Annotatef(err, "failed to describe key, id=%s", keyID)
	}

	pubKey := ""
	if includePublic {
		pub, err := p.kmsClient.GetPublicKey(&kms.GetPublicKeyInput{KeyId: &keyID})
		if err != nil {
			return errors.Annotatef(err, "failed to get public key, id=%s", keyID)
		}
		pubKey = base64.StdEncoding.EncodeToString(pub.PublicKey)
	}

	err = keyInfoFunc(
		keyID,
		aws.StringValue(resp.KeyMetadata.Description),
		aws.StringValue(resp.KeyMetadata.KeyUsage),
		aws.StringValue(resp.KeyMetadata.Origin),
		aws.StringValue(resp.KeyMetadata.KeyState),
		pubKey,
		nil,
	)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// ExportKey returns PKCS#11 URI for specified key ID.
// It does not return key bytes
func (p *Provider) ExportKey(keyID string) (string, []byte, error) {
	resp, err := p.kmsClient.DescribeKey(&kms.DescribeKeyInput{KeyId: &keyID})
	if err != nil {
		return "", nil, errors.Annotatef(err, "failed to describe key, id=%s", keyID)
	}

	uri := fmt.Sprintf("pkcs11:manufacturer=%s;id=%s;serial=%s;type=private",
		ProviderName,
		keyID,
		aws.StringValue(resp.KeyMetadata.Arn),
	)

	return uri, []byte(uri), nil
}

// FindKeyPairOnSlot retrieves a previously created asymmetric key, using a specified slot.
func (p *Provider) FindKeyPairOnSlot(slotID uint, keyID, label string) (crypto.PrivateKey, error) {
	return nil, errors.Errorf("unsupported command for this crypto provider")
}

// Close allocated resources and file reloader
func (p *Provider) Close() error {
	return nil
}

// KmsLoader provides loader for KMS provider
func KmsLoader(tc cryptoprov.TokenConfig) (cryptoprov.Provider, error) {
	p, err := Init(tc)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return p, nil
}
