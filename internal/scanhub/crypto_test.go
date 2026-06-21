package scanhub

import (
	"bytes"
	"testing"
)

func TestEncryptRoundTrip(t *testing.T) {
	token := "test-hub-token-12345"
	plain := []byte(`{"t":"found","domain":"example.com","contentB64":"ZGF0YQ=="}`)
	wire, err := Encrypt(token, plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(wire, []byte("example.com")) {
		t.Fatal("plaintext visível no wire")
	}
	out, err := Decrypt(token, wire)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("roundtrip mismatch: %q", out)
	}
}

func TestDecodeFound(t *testing.T) {
	raw := []byte(`{"t":"found","domain":"a.com","path":"/.env","confidence":"HIGH","contentB64":"KEY=1"}`)
	m, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.Type != TypeFound || m.Domain != "a.com" {
		t.Fatalf("unexpected: %+v", m)
	}
}
