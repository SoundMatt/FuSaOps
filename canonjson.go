package fusaops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// Canonicalize re-serializes JSON data per a practical subset of RFC 8785
// (the JSON Canonicalization Scheme): object keys sorted, no insignificant
// whitespace, numbers in shortest round-trip form. It targets the JSON
// shapes this codebase emits (structs marshaled via encoding/json) rather
// than adversarial input — object keys are sorted by Go's default string
// comparison, which matches RFC 8785's UTF-16 code-unit ordering except for
// keys containing supplementary-plane codepoints, which do not occur in any
// x-FuSa document.
//
//fusa:req REQ-FO-CORE008
func Canonicalize(data []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("canonjson: decode: %w", err)
	}
	var buf bytes.Buffer
	if err := writeCanonical(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeCanonical(buf *bytes.Buffer, v interface{}) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case json.Number:
		buf.WriteString(canonicalNumber(x))
	case string:
		buf.Write(canonicalString(x))
	case []interface{}:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]interface{}:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.Write(canonicalString(k))
			buf.WriteByte(':')
			if err := writeCanonical(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		return fmt.Errorf("canonjson: unsupported type %T", v)
	}
	return nil
}

// canonicalNumber formats n in the shortest round-trip form: integers with
// no decimal point or exponent, everything else via Go's shortest-round-trip
// float formatting.
func canonicalNumber(n json.Number) string {
	if i, err := n.Int64(); err == nil {
		return strconv.FormatInt(i, 10)
	}
	f, err := n.Float64()
	if err != nil {
		return n.String()
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// canonicalString encodes s as a JSON string with HTML-escaping disabled, so
// the output matches standard JSON string escaping rather than Go's
// web-safe default (which would otherwise escape '<', '>', '&').
func canonicalString(s string) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}
