// Package push sends Web Push messages. Phase 1 only creates the VAPID keys.
package push

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// KeyFile is the VAPID private key inside the data folder.
const KeyFile = "vapid-private.pem"

// Keys is the server's VAPID key pair.
type Keys struct {
	Private *ecdsa.PrivateKey
}

// PublicKey is the uncompressed P-256 point, base64url, as browsers expect
// for applicationServerKey.
func (k Keys) PublicKey() (string, error) {
	pub, err := k.Private.PublicKey.ECDH()
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(pub.Bytes()), nil
}

// LoadOrCreateKeys reads the key from dataDir, creating it on first run.
func LoadOrCreateKeys(dataDir string) (Keys, error) {
	path := filepath.Join(dataDir, KeyFile)
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return createKeys(path)
	}
	if err != nil {
		return Keys{}, fmt.Errorf("read VAPID key: %w", err)
	}

	block, _ := pem.Decode(b)
	if block == nil || block.Type != "PRIVATE KEY" {
		return Keys{}, fmt.Errorf("%s is not a PEM private key", path)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return Keys{}, fmt.Errorf("parse VAPID key: %w", err)
	}
	ec, ok := key.(*ecdsa.PrivateKey)
	if !ok || ec.Curve != elliptic.P256() {
		return Keys{}, fmt.Errorf("%s is not a P-256 key", path)
	}
	return Keys{Private: ec}, nil
}

func createKeys(path string) (Keys, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return Keys{}, err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return Keys{}, err
	}
	// O_EXCL: never overwrite a key that appeared since the read; every
	// subscription would break with a new one.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return Keys{}, fmt.Errorf("create VAPID key: %w", err)
	}
	if err := pem.Encode(f, &pem.Block{Type: "PRIVATE KEY", Bytes: der}); err != nil {
		_ = f.Close()
		return Keys{}, fmt.Errorf("write VAPID key: %w", err)
	}
	if err := f.Close(); err != nil {
		return Keys{}, fmt.Errorf("write VAPID key: %w", err)
	}
	return Keys{Private: key}, nil
}
