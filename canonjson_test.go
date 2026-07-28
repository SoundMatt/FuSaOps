package fusaops

import (
	"bytes"
	"testing"
)

//fusa:test REQ-FO-CORE008
func TestCanonicalizeSortsKeys(t *testing.T) {
	got, err := Canonicalize([]byte(`{"b":1,"a":2,"c":{"z":1,"y":2}}`))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	want := `{"a":2,"b":1,"c":{"y":2,"z":1}}`
	if string(got) != want {
		t.Errorf("Canonicalize() = %s, want %s", got, want)
	}
}

//fusa:test REQ-FO-CORE008
func TestCanonicalizeStripsWhitespace(t *testing.T) {
	got, err := Canonicalize([]byte(`{
		"a" : [1, 2, 3],
		"b" : "hello world"
	}`))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	want := `{"a":[1,2,3],"b":"hello world"}`
	if string(got) != want {
		t.Errorf("Canonicalize() = %s, want %s", got, want)
	}
}

//fusa:test REQ-FO-CORE008
func TestCanonicalizeIsIdempotentAcrossKeyOrder(t *testing.T) {
	a, err := Canonicalize([]byte(`{"x":1,"y":2}`))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	b, err := Canonicalize([]byte(`{"y":2,"x":1}`))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	if string(a) != string(b) {
		t.Errorf("Canonicalize order dependence: %s != %s", a, b)
	}
}

//fusa:test REQ-FO-CORE008
func TestCanonicalizeNoHTMLEscaping(t *testing.T) {
	got, err := Canonicalize([]byte(`{"a":"<b>&\"quote\"</b>"}`))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	want := `{"a":"<b>&\"quote\"</b>"}`
	if string(got) != want {
		t.Errorf("Canonicalize() = %s, want %s", got, want)
	}
}

//fusa:test REQ-FO-CORE008
func TestCanonicalizeInvalidJSON(t *testing.T) {
	if _, err := Canonicalize([]byte(`not json`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

//fusa:test REQ-FO-CORE008
func TestCanonicalizeIntegersNoDecimal(t *testing.T) {
	got, err := Canonicalize([]byte(`{"n":42}`))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	if string(got) != `{"n":42}` {
		t.Errorf("Canonicalize() = %s, want integer with no decimal point", got)
	}
}

// TestCanonicalizeFloat covers canonicalNumber's non-integer branch
// (strconv.FormatFloat), not exercised by the integer-only cases above.
//
//fusa:test REQ-FO-CORE008
func TestCanonicalizeFloat(t *testing.T) {
	got, err := Canonicalize([]byte(`{"pct":96.3}`))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	if string(got) != `{"pct":96.3}` {
		t.Errorf("Canonicalize() = %s, want {\"pct\":96.3}", got)
	}
}

// TestCanonicalizeNestedArrayError covers writeCanonical's error-propagation
// path when an element deep inside an array/object cannot be canonicalized.
//
//fusa:test REQ-FO-CORE008
func TestCanonicalizeNestedArrayError(t *testing.T) {
	var buf bytes.Buffer
	// A Go channel has no JSON representation, so writeCanonical must return
	// an error rather than panic or silently drop the value.
	err := writeCanonical(&buf, []interface{}{make(chan int)})
	if err == nil {
		t.Error("expected error for unsupported nested type, got nil")
	}

	buf.Reset()
	err = writeCanonical(&buf, map[string]interface{}{"k": make(chan int)})
	if err == nil {
		t.Error("expected error for unsupported map-value type, got nil")
	}
}
