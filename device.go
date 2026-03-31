package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
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

	// Raw fields from the JSON file
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

	// Decode the keys from base64
	pubBytes, err := base64.RawURLEncoding.DecodeString(identity.RawPublicKey)
	if err != nil {
		// Try standard base64
		pubBytes, err = base64.StdEncoding.DecodeString(identity.RawPublicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decode public key: %w", err)
		}
	}
	identity.PublicKey = ed25519.PublicKey(pubBytes)

	privBytes, err := base64.RawURLEncoding.DecodeString(identity.RawPrivateKey)
	if err != nil {
		privBytes, err = base64.StdEncoding.DecodeString(identity.RawPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decode private key: %w", err)
		}
	}
	identity.PrivateKey = ed25519.PrivateKey(privBytes)

	return &identity, nil
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

	return map[string]any{
		"id":        d.DeviceID,
		"publicKey": base64.RawURLEncoding.EncodeToString(d.PublicKey),
		"signature": base64.RawURLEncoding.EncodeToString(signature),
		"signedAt":  now,
		"nonce":     nonce,
	}
}
