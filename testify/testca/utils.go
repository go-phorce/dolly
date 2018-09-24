package testca

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os/exec"
)

// ToPFX converts cert with private key to PFX
func ToPFX(cert *x509.Certificate, priv interface{}, password string) []byte {
	// only allow alphanumeric passwords
	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		default:
			panic("password must be alphanumeric")
		}
	}

	passout := fmt.Sprintf("pass:%s", password)
	cmd := exec.Command("openssl", "pkcs12", "-export", "-passout", passout)

	cmd.Stdin = bytes.NewReader(append(append(ToPKCS8(priv), '\n'), ToPEM(cert)...))

	out := new(bytes.Buffer)
	cmd.Stdout = out

	if err := cmd.Run(); err != nil {
		panic(err)
	}

	return out.Bytes()
}

// ToPEM exports cert to PEM
func ToPEM(cert *x509.Certificate) []byte {
	buf := new(bytes.Buffer)
	if err := pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

// ToDER exports private key to DER
func ToDER(priv interface{}) []byte {
	var (
		der []byte
		err error
	)
	switch p := priv.(type) {
	case *rsa.PrivateKey:
		der = x509.MarshalPKCS1PrivateKey(p)
	case *ecdsa.PrivateKey:
		der, err = x509.MarshalECPrivateKey(p)
	default:
		err = errors.New("unknown key type")
	}
	if err != nil {
		panic(err)
	}

	return der
}

// PrivKeyToPEM exports private key to PEM
func PrivKeyToPEM(priv interface{}) []byte {
	var (
		pemKey []byte
		err    error
	)
	switch key := priv.(type) {
	case *rsa.PrivateKey:
		der := x509.MarshalPKCS1PrivateKey(key)
		pemKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	case *ecdsa.PrivateKey:
		der, _ := x509.MarshalECPrivateKey(key)
		pemKey = pem.EncodeToMemory(&pem.Block{Type: "ECDSA PRIVATE KEY", Bytes: der})
	default:
		err = errors.New("unknown key type")
	}
	if err != nil {
		panic(err)
	}

	return pemKey
}

// ToPKCS8 exports private key to PKCS8
func ToPKCS8(priv interface{}) []byte {
	cmd := exec.Command("openssl", "pkcs8", "-topk8", "-nocrypt", "-inform", "DER")

	cmd.Stdin = bytes.NewReader(ToDER(priv))

	out := new(bytes.Buffer)
	cmd.Stdout = out

	if err := cmd.Run(); err != nil {
		panic(err)
	}

	return out.Bytes()
}
