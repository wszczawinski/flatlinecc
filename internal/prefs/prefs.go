// Package prefs edits the Wahoo app's SharedPreferences config, BCfgManager-Bolt.xml
// (see docs/format.md). Values are replaced in place; the rest of the file is untouched.
package prefs

import (
	"bytes"
	"fmt"
	"slices"
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
	return slices.Concat(doc[:start], []byte("["+strings.Join(parts, ",")+"]"), doc[end:]), nil
}

// Bool returns a boolean field, stored as <boolean ... value="true" />.
func Bool(doc []byte, field int) (bool, error) {
	start, end, err := locateBool(doc, field)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(string(doc[start:end]))
}

// SetBool returns a copy of doc with the boolean field's value replaced.
func SetBool(doc []byte, field int, value bool) ([]byte, error) {
	start, end, err := locateBool(doc, field)
	if err != nil {
		return nil, err
	}
	return slices.Concat(doc[:start], []byte(strconv.FormatBool(value)), doc[end:]), nil
}

// key is the name of a field's entry without a profile.
func key(field int) string {
	return fmt.Sprintf(`name="{&quot;type&quot;:%d,&quot;boltId&quot;:&quot;&quot;}"`, field)
}

// locate returns the value bounds of the field's <string> entry without a profile.
func locate(doc []byte, field int) (start, end int, err error) {
	return between(doc, field, "<string "+key(field)+">", "</string>")
}

// locateBool returns the value bounds of the field's <boolean> entry without a profile.
func locateBool(doc []byte, field int) (start, end int, err error) {
	return between(doc, field, "<boolean "+key(field)+` value="`, `"`)
}

// between returns the bounds of the text between prefix and the next suffix.
func between(doc []byte, field int, prefix, suffix string) (start, end int, err error) {
	i := bytes.Index(doc, []byte(prefix))
	if i < 0 {
		return 0, 0, fmt.Errorf("field %d not found", field)
	}
	start = i + len(prefix)
	n := bytes.Index(doc[start:], []byte(suffix))
	if n < 0 {
		return 0, 0, fmt.Errorf("field %d: unterminated value", field)
	}
	return start, start + n, nil
}
