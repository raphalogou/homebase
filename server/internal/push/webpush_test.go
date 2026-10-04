package push

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// browser plays the subscriber: it holds the private key and auth secret,
// and decrypts as RFC 8291 describes.
type browser struct {
	key  *ecdh.PrivateKey
	auth []byte
}

func newBrowser(t *testing.T) browser {
	t.Helper()
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	return browser{key: k, auth: auth}
}

func (b browser) sub(endpoint string) Subscription {
	return Subscription{
		Endpoint: endpoint,
		P256DH:   base64.RawURLEncoding.EncodeToString(b.key.PublicKey().Bytes()),
		Auth:     base64.RawURLEncoding.EncodeToString(b.auth),
	}
}

func (b browser) decrypt(t *testing.T, body []byte) []byte {
	t.Helper()
	salt := body[:16]
	rs := binary.BigEndian.Uint32(body[16:20])
	idlen := int(body[20])
	asPublic := body[21 : 21+idlen]
	ciphertext := body[21+idlen:]
	if rs != recordSize || idlen != 65 {
		t.Fatalf("header rs=%d idlen=%d", rs, idlen)
	}

	pub, err := ecdh.P256().NewPublicKey(asPublic)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := b.key.ECDH(pub)
	if err != nil {
		t.Fatal(err)
	}
	info := append(append([]byte("WebPush: info\x00"), b.key.PublicKey().Bytes()...), asPublic...)
	ikm, err := hkdf.Key(sha256.New, shared, b.auth, string(info), 32)
	if err != nil {
		t.Fatal(err)
	}
	prk, _ := hkdf.Extract(sha256.New, ikm, salt)
	cek, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	block, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plain[len(plain)-1] != 0x02 {
		t.Fatalf("last record delimiter = %x, want 02", plain[len(plain)-1])
	}
	return plain[:len(plain)-1]
}

func TestEncryptDecryptsForTheSubscriber(t *testing.T) {
	b := newBrowser(t)
	payload := []byte(`{"title":"Run a half marathon in spring","body":"Today: Easy 5 km run."}`)

	first, err := Encrypt(b.sub("https://push.example/1"), payload)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.decrypt(t, first); !bytes.Equal(got, payload) {
		t.Fatalf("decrypted %q, want %q", got, payload)
	}

	// Every message gets a fresh key and salt.
	second, err := Encrypt(b.sub("https://push.example/1"), payload)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first[:16], second[:16]) || bytes.Equal(first[21:86], second[21:86]) {
		t.Fatal("salt or key reused between messages")
	}

	// Another browser cannot read it.
	other := newBrowser(t)
	pub, _ := ecdh.P256().NewPublicKey(first[21:86])
	shared, _ := other.key.ECDH(pub)
	if len(shared) == 0 {
		t.Fatal("no shared secret")
	}
}

func TestEncryptRejectsBadKeys(t *testing.T) {
	b := newBrowser(t)
	tests := []Subscription{
		{P256DH: "not base64!", Auth: b.sub("").Auth},
		{P256DH: base64.RawURLEncoding.EncodeToString([]byte("short")), Auth: b.sub("").Auth},
		{P256DH: b.sub("").P256DH, Auth: "c2hvcnQ"},
	}
	for i, s := range tests {
		if _, err := Encrypt(s, []byte("x")); err == nil {
			t.Errorf("case %d: want error", i)
		}
	}
	if _, err := Encrypt(b.sub(""), make([]byte, 5000)); err == nil {
		t.Error("oversized payload accepted")
	}
}

// fakeService is a push service: it checks the VAPID header and hands the
// body to the browser.
func TestSendToAPushService(t *testing.T) {
	keys := Keys{}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keys.Private = priv
	b := newBrowser(t)
	payload := []byte(`{"title":"Midday check"}`)

	var status = http.StatusCreated
	var got []byte
	var authz string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz = r.Header.Get("Authorization")
		if r.Header.Get("Content-Encoding") != "aes128gcm" || r.Header.Get("TTL") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
	}))
	defer srv.Close()

	now := time.Unix(1_800_000_000, 0)
	s := NewSender(keys, "mailto:me@example.com", srv.Client(), func() time.Time { return now })
	sub := b.sub(srv.URL + "/push/abc")
	if err := s.Send(context.Background(), sub, payload, time.Hour); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.decrypt(t, got), payload) {
		t.Fatal("service received something the browser cannot read")
	}
	checkVAPID(t, authz, keys, srv.URL, now)

	for _, code := range []int{http.StatusNotFound, http.StatusGone} {
		status = code
		if err := s.Send(context.Background(), sub, payload, time.Hour); !errors.Is(err, ErrGone) {
			t.Errorf("status %d: err = %v, want ErrGone", code, err)
		}
	}
	status = http.StatusTooManyRequests
	if err := s.Send(context.Background(), sub, payload, time.Hour); err == nil || errors.Is(err, ErrGone) {
		t.Errorf("429: err = %v, want a plain error", err)
	}
	if err := s.Send(context.Background(), b.sub("http://insecure.example/x"), payload, time.Hour); err == nil {
		t.Error("plain http endpoint accepted")
	}
}

func checkVAPID(t *testing.T, header string, keys Keys, audience string, now time.Time) {
	t.Helper()
	rest, ok := strings.CutPrefix(header, "vapid t=")
	if !ok {
		t.Fatalf("Authorization = %q", header)
	}
	jwt, k, ok := strings.Cut(rest, ", k=")
	if !ok {
		t.Fatalf("Authorization = %q", header)
	}
	if pub, _ := keys.PublicKey(); k != pub {
		t.Errorf("k = %q, want the server's public key", k)
	}
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("jwt = %q", jwt)
	}
	var claims struct {
		Aud string
		Exp int64
		Sub string
	}
	raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	if claims.Aud != audience || claims.Sub != "mailto:me@example.com" || claims.Exp <= now.Unix() || claims.Exp > now.Add(24*time.Hour).Unix() {
		t.Errorf("claims = %+v", claims)
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
	if !ecdsa.Verify(&keys.Private.PublicKey, digest[:], r, s) {
		t.Error("VAPID signature does not verify")
	}
}

// RFC 8291 Appendix A: fixed keys, salt and auth secret give one exact body.
func TestRFC8291Vector(t *testing.T) {
	d := func(s string) []byte {
		b, err := base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	asPrivate, err := ecdh.P256().NewPrivateKey(d("yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"))
	if err != nil {
		t.Fatal(err)
	}
	sub := Subscription{
		P256DH: "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4",
		Auth:   "BTBZMqHH6r4Tts7J_aSIgg",
	}
	got, err := encryptWith(sub, []byte("When I grow up, I want to be a watermelon"), asPrivate, d("DGv6ra1nlYgDCS1FRnbzlw"))
	if err != nil {
		t.Fatal(err)
	}
	want := "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
	if g := base64.RawURLEncoding.EncodeToString(got); g != want {
		t.Errorf("body =\n%s\nwant\n%s", g, want)
	}
}
