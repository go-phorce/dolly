package authority

import (
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"

	"github.com/go-phorce/dolly/xpki/csr"
	"github.com/juju/errors"
)

type policyInformation struct {
	PolicyIdentifier asn1.ObjectIdentifier
	Qualifiers       []interface{} `asn1:"tag:optional,omitempty"`
}

type cpsPolicyQualifier struct {
	PolicyQualifierID asn1.ObjectIdentifier
	Qualifier         string `asn1:"tag:optional,ia5"`
}

type userNotice struct {
	ExplicitText string `asn1:"tag:optional,utf8"`
}
type userNoticePolicyQualifier struct {
	PolicyQualifierID asn1.ObjectIdentifier
	Qualifier         userNotice
}

var (
	// Per https://tools.ietf.org/html/rfc3280.html#page-106, this represents:
	// iso(1) identified-organization(3) dod(6) internet(1) security(5)
	//   mechanisms(5) pkix(7) id-qt(2) id-qt-cps(1)
	iDQTCertificationPracticeStatement = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 2, 1}
	// iso(1) identified-organization(3) dod(6) internet(1) security(5)
	//   mechanisms(5) pkix(7) id-qt(2) id-qt-unotice(2)
	iDQTUserNotice = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 2, 2}

	// CTPoisonOID is the object ID of the critical poison extension for precertificates
	// https://tools.ietf.org/html/rfc6962#page-9
	CTPoisonOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 3}

	// SCTListOID is the object ID for the Signed Certificate Timestamp certificate extension
	// https://tools.ietf.org/html/rfc6962#page-14
	SCTListOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 2}
)

// addPolicies adds Certificate Policies and optional Policy Qualifiers to a
// certificate, based on the input config. Go's x509 library allows setting
// Certificate Policies easily, but does not support nested Policy Qualifiers
// under those policies. So we need to construct the ASN.1 structure ourselves.
func addPolicies(template *x509.Certificate, policies []csr.CertificatePolicy) error {
	asn1PolicyList := []policyInformation{}

	for _, policy := range policies {
		pi := policyInformation{
			// The PolicyIdentifier is an OID assigned to a given issuer.
			PolicyIdentifier: asn1.ObjectIdentifier(policy.ID),
		}
		for _, qualifier := range policy.Qualifiers {
			switch qualifier.Type {
			case "id-qt-unotice":
				pi.Qualifiers = append(pi.Qualifiers,
					userNoticePolicyQualifier{
						PolicyQualifierID: iDQTUserNotice,
						Qualifier: userNotice{
							ExplicitText: qualifier.Value,
						},
					})
			case "id-qt-cps":
				pi.Qualifiers = append(pi.Qualifiers,
					cpsPolicyQualifier{
						PolicyQualifierID: iDQTCertificationPracticeStatement,
						Qualifier:         qualifier.Value,
					})
			default:
				return errors.New("Invalid qualifier type in Policies " + qualifier.Type)
			}
		}
		asn1PolicyList = append(asn1PolicyList, pi)
	}

	asn1Bytes, err := asn1.Marshal(asn1PolicyList)
	if err != nil {
		return errors.Trace(err)
	}

	template.ExtraExtensions = append(template.ExtraExtensions, pkix.Extension{
		Id:       asn1.ObjectIdentifier{2, 5, 29, 32},
		Critical: false,
		Value:    asn1Bytes,
	})
	return nil
}

// computeSKI derives an SKI from the certificate's public key in a
// standard manner. This is done by computing the SHA-1 digest of the
// SubjectPublicKeyInfo component of the certificate.
func computeSKI(template *x509.Certificate) ([]byte, error) {
	pub := template.PublicKey
	encodedPub, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var subPKI subjectPublicKeyInfo
	_, err = asn1.Unmarshal(encodedPub, &subPKI)
	if err != nil {
		return nil, errors.Trace(err)
	}

	pubHash := sha1.Sum(subPKI.SubjectPublicKey.Bytes)
	return pubHash[:], nil
}

type subjectPublicKeyInfo struct {
	Algorithm        pkix.AlgorithmIdentifier
	SubjectPublicKey asn1.BitString
}
