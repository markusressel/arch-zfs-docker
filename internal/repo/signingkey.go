package repo

import (
	"bufio"
	"crypto/sha1" //nolint:gosec // the OpenPGP v4 fingerprint is defined as SHA-1
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SigningKeyFile is the name of the public key the index step publishes next to the database.
func SigningKeyFile(repoName string) string { return repoName + ".asc" }

// SigningKey reads the public key published in the repository directory (<dir>/<repo>.asc) and returns
// its fingerprint. ok is false when there is none.
func SigningKey(dir, repoName string) (fingerprint string, ok bool) {
	b, err := os.ReadFile(filepath.Join(dir, SigningKeyFile(repoName)))
	if err != nil {
		return "", false
	}
	fpr, err := ArmoredKeyFingerprint(string(b))
	if err != nil {
		return "", false
	}
	return fpr, true
}

// ArmoredKeyFingerprint returns the fingerprint (upper-case hex) of the primary key of an armored
// OpenPGP public key block, for key versions 4, 5 and 6.
func ArmoredKeyFingerprint(armored string) (string, error) {
	data, err := dearmor(armored)
	if err != nil {
		return "", err
	}
	tag, body, err := firstPacket(data)
	if err != nil {
		return "", err
	}
	if tag != 6 {
		return "", fmt.Errorf("first packet is not a public key (tag %d)", tag)
	}
	if len(body) == 0 {
		return "", fmt.Errorf("empty public key packet")
	}
	var sum []byte
	switch body[0] {
	case 4:
		h := sha1.New() //nolint:gosec
		h.Write([]byte{0x99, byte(len(body) >> 8), byte(len(body))})
		h.Write(body)
		sum = h.Sum(nil)
	case 5, 6:
		prefix := byte(0x9A)
		if body[0] == 6 {
			prefix = 0x9B
		}
		h := sha256.New()
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(body)))
		h.Write([]byte{prefix})
		h.Write(n[:])
		h.Write(body)
		sum = h.Sum(nil)
	default:
		return "", fmt.Errorf("unsupported key version %d", body[0])
	}
	return strings.ToUpper(hex.EncodeToString(sum)), nil
}

// dearmor decodes the base64 body of an ASCII-armored public key block.
func dearmor(armored string) ([]byte, error) {
	sc := bufio.NewScanner(strings.NewReader(armored))
	var b64 strings.Builder
	state := 0 // 0 before the block, 1 in the headers, 2 in the body
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case state == 0 && line == "-----BEGIN PGP PUBLIC KEY BLOCK-----":
			state = 1
		case state == 1 && line == "":
			state = 2
		case state == 1 && !strings.Contains(line, ":"):
			// No headers: the body starts right away.
			state = 2
			b64.WriteString(line)
		case state == 2 && strings.HasPrefix(line, "-----END"):
			return base64.StdEncoding.DecodeString(b64.String())
		case state == 2 && strings.HasPrefix(line, "="):
			// CRC-24 checksum line.
		case state == 2:
			b64.WriteString(line)
		}
	}
	return nil, fmt.Errorf("no armored public key block")
}

// firstPacket returns tag and body of the first OpenPGP packet (old or new packet format, definite lengths).
func firstPacket(data []byte) (int, []byte, error) {
	if len(data) < 2 || data[0]&0x80 == 0 {
		return 0, nil, fmt.Errorf("not an OpenPGP packet")
	}
	var tag, length, off int
	if data[0]&0x40 != 0 { // new format
		tag = int(data[0] & 0x3f)
		switch first := int(data[1]); {
		case first < 192:
			length, off = first, 2
		case first < 224:
			if len(data) < 3 {
				return 0, nil, fmt.Errorf("truncated packet")
			}
			length, off = (first-192)<<8+int(data[2])+192, 3
		case first == 255:
			if len(data) < 6 {
				return 0, nil, fmt.Errorf("truncated packet")
			}
			length, off = int(binary.BigEndian.Uint32(data[2:6])), 6
		default:
			return 0, nil, fmt.Errorf("partial body lengths are not supported for keys")
		}
	} else { // old format
		tag = int(data[0]>>2) & 0x0f
		switch data[0] & 0x03 {
		case 0:
			length, off = int(data[1]), 2
		case 1:
			if len(data) < 3 {
				return 0, nil, fmt.Errorf("truncated packet")
			}
			length, off = int(binary.BigEndian.Uint16(data[1:3])), 3
		case 2:
			if len(data) < 5 {
				return 0, nil, fmt.Errorf("truncated packet")
			}
			length, off = int(binary.BigEndian.Uint32(data[1:5])), 5
		default:
			return 0, nil, fmt.Errorf("indeterminate packet length")
		}
	}
	if length < 0 || off+length > len(data) {
		return 0, nil, fmt.Errorf("truncated packet")
	}
	return tag, data[off : off+length], nil
}
