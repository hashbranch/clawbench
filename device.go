package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DeviceIdentity holds the Ed25519 key pair and device ID
// from ~/.openclaw/identity/device.json.
type DeviceIdentity struct {
	DeviceID   string            `json:"deviceId"`
	PublicKey  ed25519.PublicKey  `json:"-"`
	PrivateKey ed25519.PrivateKey `json:"-"`

	// Raw fields from the JSON file (PEM-encoded PKCS8)
	RawPublicKey  string `json:"publicKey"`
	RawPrivateKey string `json:"privateKey"`
}

// DeviceAuth holds the stored device token
// from ~/.openclaw/identity/device-auth.json.
type DeviceAuth struct {
	DeviceToken string   `json:"deviceToken"`
	Scopes      []string `json:"scopes"`
}

// LoadDeviceIdentity reads the OpenClaw CLI's existing device identity.
func LoadDeviceIdentity() (*DeviceIdentity, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	identityPath := filepath.Join(home, ".openclaw", "identity", "device.json")
	data, err := os.ReadFile(identityPath)
	if err != nil {
		return nil, fmt.Errorf("device identity not found at %s: %w (has openclaw CLI been set up?)", identityPath, err)
	}

	var identity DeviceIdentity
	if err := json.Unmarshal(data, &identity); err != nil {
		return nil, fmt.Errorf("failed to parse device identity: %w", err)
	}

	// Parse private key (PEM-encoded PKCS8)
	privKey, err := parseEdPrivateKey(identity.RawPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	identity.PrivateKey = privKey

	// Parse public key (PEM-encoded SPKI)
	pubKey, err := parseEdPublicKey(identity.RawPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	identity.PublicKey = pubKey

	return &identity, nil
}

// parseEdPrivateKey handles PEM PKCS8 or raw base64 encoded Ed25519 private keys.
func parseEdPrivateKey(raw string) (ed25519.PrivateKey, error) {
	// Try PEM first
	block, _ := pem.Decode([]byte(raw))
	if block != nil {
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("PKCS8 parse failed: %w", err)
		}
		edKey, ok := parsed.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not Ed25519")
		}
		return edKey, nil
	}

	// Fallback: try raw base64
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("not PEM and not valid base64")
		}
	}

	switch len(decoded) {
	case 64:
		return ed25519.PrivateKey(decoded), nil
	case 32:
		return ed25519.NewKeyFromSeed(decoded), nil
	case 48:
		// Raw PKCS8 DER without PEM wrapper, seed is last 32 bytes
		seed := decoded[len(decoded)-32:]
		return ed25519.NewKeyFromSeed(seed), nil
	default:
		// Try parsing as DER
		parsed, err := x509.ParsePKCS8PrivateKey(decoded)
		if err != nil {
			return nil, fmt.Errorf("unexpected key length %d and DER parse failed: %w", len(decoded), err)
		}
		edKey, ok := parsed.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not Ed25519")
		}
		return edKey, nil
	}
}

// parseEdPublicKey handles PEM SPKI or raw base64 encoded Ed25519 public keys.
func parseEdPublicKey(raw string) (ed25519.PublicKey, error) {
	// Try PEM first
	block, _ := pem.Decode([]byte(raw))
	if block != nil {
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("SPKI parse failed: %w", err)
		}
		edKey, ok := parsed.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not Ed25519")
		}
		return edKey, nil
	}

	// Fallback: try raw base64
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("not PEM and not valid base64")
		}
	}

	switch len(decoded) {
	case 32:
		return ed25519.PublicKey(decoded), nil
	case 44:
		// Raw SPKI DER without PEM wrapper, public key is last 32 bytes
		return ed25519.PublicKey(decoded[len(decoded)-32:]), nil
	default:
		// Try parsing as DER
		parsed, err := x509.ParsePKIXPublicKey(decoded)
		if err != nil {
			return nil, fmt.Errorf("unexpected key length %d and DER parse failed: %w", len(decoded), err)
		}
		edKey, ok := parsed.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not Ed25519")
		}
		return edKey, nil
	}
}

// LoadDeviceAuth reads the stored device token.
func LoadDeviceAuth() (*DeviceAuth, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	authPath := filepath.Join(home, ".openclaw", "identity", "device-auth.json")
	data, err := os.ReadFile(authPath)
	if err != nil {
		return nil, nil // device-auth is optional
	}

	var auth DeviceAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, nil // non-fatal
	}
	return &auth, nil
}

// SignChallenge signs the connect.challenge nonce using the v3 payload format.
// Returns the device auth payload for the connect handshake.
func (d *DeviceIdentity) SignChallenge(nonce string) map[string]any {
	now := time.Now().UnixMilli()

	// v3 payload: "openclaw-device-auth-v3:<nonce>:<timestamp>"
	payload := fmt.Sprintf("openclaw-device-auth-v3:%s:%d", nonce, now)
	signature := ed25519.Sign(d.PrivateKey, []byte(payload))

	// Send raw 32-byte public key as base64url (not SPKI)
	rawPub := d.PublicKey
	if len(rawPub) > 32 {
		rawPub = rawPub[len(rawPub)-32:]
	}

	return map[string]any{
		"id":        d.DeviceID,
		"publicKey": base64.RawURLEncoding.EncodeToString(rawPub),
		"signature": base64.RawURLEncoding.EncodeToString(signature),
		"signedAt":  now,
		"nonce":     nonce,
	}
}
