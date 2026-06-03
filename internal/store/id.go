package store

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

func newID(prefix string) string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return prefix + "_" + strings.ReplaceAll(base64.RawURLEncoding.EncodeToString([]byte(prefix)), "-", "")
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(buf[:])
}
