package push

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"time"

	"crypto/hkdf"
)

// Subscription is what the browser's PushManager hands out.
type Subscription struct {
	Endpoint string
	P256DH   string // base64url, the browser's P-256 public key
	Auth     string // base64url, 16 random bytes
}

// ErrGone means the push service no longer knows the subscription (404 or
// 410); the caller deletes it.
var ErrGone = errors.New("subscription gone")

// recordSize is the rs field of the aes128gcm header; one record is enough
// for a Web Push payload, which is at most about 4 KB.
const recordSize = 4096

// Encrypt seals payload for one subscriber as an aes128gcm body (RFC 8291,
// RFC 8188), with a fresh key pair and salt for every message.
func Encrypt(sub Subscription, payload []byte) ([]byte, error) {
	return encrypt(sub, payload, rand.Reader)
}

func encrypt(sub Subscription, payload []byte, random io.Reader) ([]byte, error) {
	asPrivate, err := ecdh.P256().GenerateKey(random)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if _, err := io.ReadFull(random, salt); err != nil {
		return nil, err
	}
	return encryptWith(sub, payload, asPrivate, salt)
}

// encryptWith is Encrypt with the sender's key pair and salt given, so the
// RFC 8291 test vector can be checked exactly.
func encryptWith(sub Subscription, payload []byte, asPrivate *ecdh.PrivateKey, salt []byte) ([]byte, error) {
	uaBytes, err := base64.RawURLEncoding.DecodeString(trimPad(sub.P256DH))
	if err != nil {
		return nil, fmt.Errorf("p256dh: %w", err)
	}
	authSecret, err := base64.RawURLEncoding.DecodeString(trimPad(sub.Auth))
	if err != nil || len(authSecret) < 16 {
		return nil, errors.New("auth secret must be 16 bytes")
	}
	uaPublic, err := ecdh.P256().NewPublicKey(uaBytes)
	if err != nil {
		return nil, fmt.Errorf("p256dh: %w", err)
	}
	if len(payload) > recordSize-16-1-86 {
		return nil, errors.New("payload too large for one record")
	}

	asPublic := asPrivate.PublicKey().Bytes()
	shared, err := asPrivate.ECDH(uaPublic)
	if err != nil {
		return nil, err
	}

	keyInfo := append(append([]byte("WebPush: info\x00"), uaBytes...), asPublic...)
	ikm, err := hkdf.Key(sha256.New, shared, authSecret, string(keyInfo), 32)
	if err != nil {
		return nil, err
	}
	cek, nonce, err := contentKeys(sha256.New, ikm, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// 0x02 marks the last (and only) record; no padding.
	plaintext := append(append([]byte{}, payload...), 0x02)
	sealed := gcm.Seal(nil, nonce, plaintext, nil)

	var body bytes.Buffer
	body.Write(salt)
	_ = binary.Write(&body, binary.BigEndian, uint32(recordSize))
	body.WriteByte(byte(len(asPublic)))
	body.Write(asPublic)
	body.Write(sealed)
	return body.Bytes(), nil
}

func contentKeys[H hash.Hash](h func() H, ikm, salt []byte) (cek, nonce []byte, err error) {
	prk, err := hkdf.Extract(h, ikm, salt)
	if err != nil {
		return nil, nil, err
	}
	if cek, err = hkdf.Expand(h, prk, "Content-Encoding: aes128gcm\x00", 16); err != nil {
		return nil, nil, err
	}
	nonce, err = hkdf.Expand(h, prk, "Content-Encoding: nonce\x00", 12)
	return cek, nonce, err
}

// Browsers send keys in base64url, some with padding.
func trimPad(s string) string {
	for len(s) > 0 && s[len(s)-1] == '=' {
		s = s[:len(s)-1]
	}
	return s
}

// Sender delivers messages to push services.
type Sender struct {
	keys    Keys
	subject string
	client  *http.Client
	now     func() time.Time
}

// NewSender returns a Sender. subject is HOMEBASE_VAPID_SUBJECT.
func NewSender(keys Keys, subject string, client *http.Client, now func() time.Time) *Sender {
	return &Sender{keys: keys, subject: subject, client: client, now: now}
}

// Send encrypts payload and posts it to the subscription's push service.
// ttl is how long the service may hold it for an offline device.
func (s *Sender) Send(ctx context.Context, sub Subscription, payload []byte, ttl time.Duration) error {
	u, err := url.Parse(sub.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("endpoint must be an https URL")
	}
	body, err := Encrypt(sub, payload)
	if err != nil {
		return err
	}
	jwt, err := s.vapidJWT(u.Scheme + "://" + u.Host)
	if err != nil {
		return err
	}
	pub, err := s.keys.PublicKey()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("TTL", fmt.Sprint(int(ttl.Seconds())))
	req.Header.Set("Urgency", "normal")
	req.Header.Set("Authorization", "vapid t="+jwt+", k="+pub)

	res, err := s.client.Do(req)
	if err != nil {
		// url.Error repeats the endpoint, which is a secret; keep only the cause.
		var ue *url.Error
		if errors.As(err, &ue) {
			return fmt.Errorf("push request failed: %w", ue.Err)
		}
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	switch {
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone:
		return ErrGone
	case res.StatusCode >= 300:
		return fmt.Errorf("push service answered %s", res.Status)
	}
	return nil
}

// vapidJWT signs the ES256 token of RFC 8292 for one push service origin.
func (s *Sender) vapidJWT(audience string) (string, error) {
	enc := base64.RawURLEncoding
	header := enc.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`))
	claims, err := json.Marshal(map[string]any{
		"aud": audience,
		"exp": s.now().Add(12 * time.Hour).Unix(),
		"sub": s.subject,
	})
	if err != nil {
		return "", err
	}
	signing := header + "." + enc.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signing))
	r, sv, err := ecdsa.Sign(rand.Reader, s.keys.Private, digest[:])
	if err != nil {
		return "", err
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	sv.FillBytes(sig[32:])
	return signing + "." + enc.EncodeToString(sig), nil
}
