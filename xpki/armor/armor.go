// Package armor implements OpenPGP ASCII Armor, see RFC 4880. OpenPGP Armor is
// very similar to PEM except that it has an additional CRC checksum.
package armor

import (
	"bytes"
	"encoding/base64"

	"github.com/go-phorce/dolly/xlog"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xpki", "armor")

// A Block represents an OpenPGP armored structure.
//
// The encoded form is:
//    -----BEGIN Type-----
//    Headers
//
//    base64-encoded Bytes
//    '=' base64 encoded checksum
//    -----END Type-----
// where Headers is a possibly empty sequence of Key: Value lines.
type Block struct {
	Type    string            // The type, taken from the preamble (i.e. "RSA PRIVATE KEY").
	Headers map[string]string // Optional headers.
	Bytes   []byte            // The decoded bytes of the contents. Typically a DER encoded ASN.1 structure.
	CRC     uint32
}

// getLine results the first \r\n or \n delineated line from the given byte
// array. The line does not include trailing whitespace or the trailing new
// line bytes. The remainder of the byte array (also not including the new line
// bytes) is also returned and this will always be smaller than the original
// argument.
func getLine(data []byte) (line, rest []byte) {
	i := bytes.IndexByte(data, '\n')
	var j int
	if i < 0 {
		i = len(data)
		j = i
	} else {
		j = i + 1
		if i > 0 && data[i-1] == '\r' {
			i--
		}
	}
	return bytes.TrimRight(data[0:i], " \t"), data[j:]
}

// removeWhitespace returns a copy of its input with all spaces, tab and
// newline characters removed.
func removeWhitespace(data []byte) []byte {
	result := make([]byte, len(data))
	n := 0

	for _, b := range data {
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			continue
		}
		result[n] = b
		n++
	}

	return result[0:n]
}

var pemStart = []byte("\n-----BEGIN ")
var pemEnd = []byte("\n-----END ")
var pemEndOfLine = []byte("-----")

// Decode will find the next PEM formatted block (certificate, private key
// etc) in the input. It returns that block and the remainder of the input. If
// no PEM data is found, p is nil and the whole of the input is returned in
// rest.
func Decode(data []byte) (p *Block, rest []byte) {
	var err error
	// pemStart begins with a newline. However, at the very beginning of
	// the byte array, we'll accept the start string without it.
	rest = data
	if bytes.HasPrefix(data, pemStart[1:]) {
		rest = rest[len(pemStart)-1 : len(data)]
	} else if i := bytes.Index(data, pemStart); i >= 0 {
		rest = rest[i+len(pemStart) : len(data)]
	} else {
		logger.Debug("reason=prefix_not_found")
		return nil, data
	}

	typeLine, rest := getLine(rest)
	if !bytes.HasSuffix(typeLine, pemEndOfLine) {
		logger.Debug("reason=sufix_not_found")
		return decodeError(data, rest)
	}
	typeLine = typeLine[0 : len(typeLine)-len(pemEndOfLine)]

	p = &Block{
		Headers: make(map[string]string),
		Type:    string(typeLine),
	}

	for {
		// This loop terminates because getLine's second result is
		// always smaller than its argument.
		if len(rest) == 0 {
			return nil, data
		}
		line, next := getLine(rest)

		i := bytes.IndexByte(line, ':')
		if i == -1 {
			break
		}

		// TODO(agl): need to cope with values that spread across lines.
		key, val := line[:i], line[i+1:]
		key = bytes.TrimSpace(key)
		val = bytes.TrimSpace(val)
		p.Headers[string(key)] = string(val)
		rest = next
	}

	var endIndex, endTrailerIndex int

	// If there were no headers, the END line might occur
	// immediately, without a leading newline.
	if len(p.Headers) == 0 && bytes.HasPrefix(rest, pemEnd[1:]) {
		endIndex = 0
		endTrailerIndex = len(pemEnd) - 1
	} else {
		endIndex = bytes.Index(rest, pemEnd)
		endTrailerIndex = endIndex + len(pemEnd)
	}

	if endIndex < 0 {
		logger.Debug("reason=end_index, endIndex=%d", endIndex)
		return decodeError(data, rest)
	}

	// After the "-----" of the ending line, there should be the same type
	// and then a final five dashes.
	endTrailer := rest[endTrailerIndex:]
	endTrailerLen := len(typeLine) + len(pemEndOfLine)
	if len(endTrailer) < endTrailerLen {
		logger.Debug("reason=end_trailer, endTrailerLen=%d", endTrailerLen)
		return decodeError(data, rest)
	}

	restOfEndLine := endTrailer[endTrailerLen:]
	endTrailer = endTrailer[:endTrailerLen]
	if !bytes.HasPrefix(endTrailer, typeLine) ||
		!bytes.HasSuffix(endTrailer, pemEndOfLine) {
		return decodeError(data, rest)
	}

	// The line must end with only whitespace.
	if s, _ := getLine(restOfEndLine); len(s) != 0 {
		return decodeError(data, rest)
	}

	// extract CRC bytes
	base64Block := removeWhitespace(rest[:endIndex])
	blockLen := len(base64Block)
	if blockLen < 5 || base64Block[blockLen-5] != '=' {
		logger.Debugf("reason=crc, blockLen=%d", blockLen)
		return decodeError(data, rest)
	}

	base64Data := removeWhitespace(base64Block[:blockLen-5])
	crcData := base64Block[blockLen-4:]

	// This is the checksum line
	var expectedBytes [3]byte
	var m int
	m, err = base64.StdEncoding.Decode(expectedBytes[0:], crcData)
	if m != 3 || err != nil {
		logger.Debugf("reason=crc, crc_len=%d, err=[%v]", m, err)
		return decodeError(data, rest)
	}
	p.CRC = uint32(expectedBytes[0])<<16 | uint32(expectedBytes[1])<<8 | uint32(expectedBytes[2])
	p.Bytes = make([]byte, base64.StdEncoding.DecodedLen(len(base64Data)))
	n, err := base64.StdEncoding.Decode(p.Bytes, base64Data)
	if err != nil {
		logger.Debugf("reason=base64, err=[%v]", err)
		return decodeError(data, rest)
	}
	p.Bytes = p.Bytes[:n]

	crc := uint32(crc24(crc24Init, p.Bytes) & crc24Mask)
	if p.CRC != crc {
		logger.Debugf("reason=CRC, expected=%d, actual=%d", p.CRC, crc)
		return decodeError(data, rest)
	}

	// the -1 is because we might have only matched pemEnd without the
	// leading newline if the PEM block was empty.
	_, rest = getLine(rest[endIndex+len(pemEnd)-1:])

	return
}

func decodeError(data, rest []byte) (*Block, []byte) {
	// If we get here then we have rejected a likely looking, but
	// ultimately invalid PEM block. We need to start over from a new
	// position. We have consumed the preamble line and will have consumed
	// any lines which could be header lines. However, a valid preamble
	// line is not a valid header line, therefore we cannot have consumed
	// the preamble line for the any subsequent block. Thus, we will always
	// find any valid block, no matter what bytes precede it.
	//
	// For example, if the input is
	//
	//    -----BEGIN MALFORMED BLOCK-----
	//    junk that may look like header lines
	//   or data lines, but no END line
	//
	//    -----BEGIN ACTUAL BLOCK-----
	//    realdata
	//    -----END ACTUAL BLOCK-----
	//
	// we've failed to parse using the first BEGIN line
	// and now will try again, using the second BEGIN line.
	p, rest := Decode(rest)
	if p == nil {
		rest = data
	}
	return p, rest
}

const crc24Init = 0xb704ce
const crc24Poly = 0x1864cfb
const crc24Mask = 0xffffff

// crc24 calculates the OpenPGP checksum as specified in RFC 4880, section 6.1
func crc24(crc uint32, d []byte) uint32 {
	for _, b := range d {
		crc ^= uint32(b) << 16
		for i := 0; i < 8; i++ {
			crc <<= 1
			if crc&0x1000000 != 0 {
				crc ^= crc24Poly
			}
		}
	}
	return crc
}
