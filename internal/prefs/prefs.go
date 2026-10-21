// Package prefs edits the Wahoo app's SharedPreferences config, BCfgManager-Bolt.xml
// (see docs/format.md). Values are replaced in place; the rest of the file is untouched.
package prefs

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Bytes returns a byte-array field, stored as a string like "[1,-59,0]".
func Bytes(doc []byte, field int) ([]byte, error) {
	start, end, err := locate(doc, field)
	if err != nil {
		return nil, err
	}
	raw := strings.Trim(string(doc[start:end]), "[]")
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]byte, len(parts))
	for i, p := range parts {
		n, err := strconv.ParseInt(p, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("field %d byte %d: %w", field, i, err)
		}
		out[i] = byte(n)
	}
	return out, nil
}

// SetBytes returns a copy of doc with the field's value replaced.
func SetBytes(doc []byte, field int, value []byte) ([]byte, error) {
	start, end, err := locate(doc, field)
	if err != nil {
		return nil, err
	}
	parts := make([]string, len(value))
	for i, b := range value {
		parts[i] = strconv.Itoa(int(int8(b)))
	}
	out := append([]byte{}, doc[:start]...)
	out = append(out, "["+strings.Join(parts, ",")+"]"...)
	return append(out, doc[end:]...), nil
}

// locate returns the value bounds of the field's <string> entry without a profile.
func locate(doc []byte, field int) (start, end int, err error) {
	open := fmt.Sprintf(`<string name="{&quot;type&quot;:%d,&quot;boltId&quot;:&quot;&quot;}">`, field)
	i := bytes.Index(doc, []byte(open))
	if i < 0 {
		return 0, 0, fmt.Errorf("field %d not found", field)
	}
	start = i + len(open)
	n := bytes.Index(doc[start:], []byte("</string>"))
	if n < 0 {
		return 0, 0, fmt.Errorf("field %d: unterminated value", field)
	}
	return start, start + n, nil
}
