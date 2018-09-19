package oid

// OIDStrToInfo provides mapping from OID string to Info
var OIDStrToInfo = map[string]Info{
	"1.2.840.113549.1.1.1":    RSA,
	"1.2.840.10045.2.1":       ECDSA,
	"1.3.14.3.2.26":           SHA1,
	"2.16.840.1.101.3.4.2.1":  SHA256,
	"2.16.840.1.101.3.4.2.2":  SHA384,
	"2.16.840.1.101.3.4.2.3":  SHA512,
	"2.16.840.1.101.3.4.2.7":  SHA3x224,
	"2.16.840.1.101.3.4.2.8":  SHA3x256,
	"2.16.840.1.101.3.4.2.9":  SHA3x384,
	"2.16.840.1.101.3.4.2.10": SHA3x512,
	"2.16.840.1.101.3.4.2.11": SHAKE128,
	"2.16.840.1.101.3.4.2.12": SHAKE256,
	"1.2.840.113549.1.1.5":    RSAWithSHA1,
	"1.2.840.113549.1.1.11":   RSAWithSHA256,
	"1.2.840.113549.1.1.12":   RSAWithSHA384,
	"1.2.840.113549.1.1.13":   RSAWithSHA512,
	"1.2.840.10045.4.1":       ECDSAWithSHA1,
	"1.2.840.10045.4.3.2":     ECDSAWithSHA256,
	"1.2.840.10045.4.3.3":     ECDSAWithSHA384,
	"1.2.840.10045.4.3.4":     ECDSAWithSHA512,
}

// AlgNameToInfo provides mapping from algorith name to Info
var AlgNameToInfo = map[string]Info{
	"RSA":             RSA,
	"ECDSA":           ECDSA,
	"SHA1":            SHA1,
	"SHA256":          SHA256,
	"SHA384":          SHA384,
	"SHA512":          SHA512,
	"SHA3x224":        SHA3x224,
	"SHA3-224":        SHA3x224,
	"SHA3x256":        SHA3x256,
	"SHA3-256":        SHA3x256,
	"SHA3x384":        SHA3x384,
	"SHA3-384":        SHA3x384,
	"SHA3x512":        SHA3x512,
	"SHA3-512":        SHA3x512,
	"SHAKE128":        SHAKE128,
	"SHAKE-128":       SHAKE128,
	"SHAKE256":        SHAKE256,
	"SHAKE-256":       SHAKE256,
	"RSAWithSHA1":     RSAWithSHA1,
	"RSA-SHA1":        RSAWithSHA1,
	"RSA_SHA1":        RSAWithSHA1,
	"RSAWithSHA256":   RSAWithSHA256,
	"RSA-SHA256":      RSAWithSHA256,
	"RSA_SHA256":      RSAWithSHA256,
	"RSAWithSHA384":   RSAWithSHA384,
	"RSA-SHA384":      RSAWithSHA384,
	"RSA_SHA384":      RSAWithSHA384,
	"RSAWithSHA512":   RSAWithSHA512,
	"RSA-SHA512":      RSAWithSHA512,
	"RSA_SHA512":      RSAWithSHA512,
	"ECDSAWithSHA1":   ECDSAWithSHA1,
	"ECDSA-SHA1":      ECDSAWithSHA1,
	"ECDSA_SHA1":      ECDSAWithSHA1,
	"ECDSAWithSHA256": ECDSAWithSHA256,
	"ECDSA-SHA256":    ECDSAWithSHA256,
	"ECDSA_SHA256":    ECDSAWithSHA256,
	"ECDSAWithSHA384": ECDSAWithSHA384,
	"ECDSA-SHA384":    ECDSAWithSHA384,
	"ECDSA_SHA384":    ECDSAWithSHA384,
	"ECDSAWithSHA512": ECDSAWithSHA512,
	"ECDSA-SHA512":    ECDSAWithSHA512,
	"ECDSA_SHA512":    ECDSAWithSHA512,
}

// LookupByOID returns an algorithm by OID
func LookupByOID(oid string) Info {
	if len(oid) == 0 {
		return nil
	}
	return OIDStrToInfo[oid]
}

// LookupByName returns an algorithm by name
func LookupByName(name string) Info {
	if len(name) == 0 {
		return nil
	}
	return AlgNameToInfo[name]
}
