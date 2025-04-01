package recorder

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

// SecurityOptions configures security features for event recording
type SecurityOptions struct {
	// Encryption settings
	EnableEncryption bool
	EncryptionKey    []byte // Should be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256

	// Redaction settings
	EnableRedaction      bool
	RedactionPatterns    []string // Regex patterns to identify sensitive data
	RedactionReplacement string   // String to replace sensitive data with

	// Integrity verification settings
	EnableIntegrityCheck bool
	IntegrityKey         []byte // Key for HMAC
}

// DefaultSecurityOptions returns the default security options (no security features enabled)
func DefaultSecurityOptions() SecurityOptions {
	return SecurityOptions{
		EnableEncryption:     false,
		EncryptionKey:        nil,
		EnableRedaction:      false,
		RedactionPatterns:    []string{"password", "token", "secret", "key", "credential"},
		RedactionReplacement: "***REDACTED***",
		EnableIntegrityCheck: false,
		IntegrityKey:         nil,
	}
}

// WithEncryption enables encryption with the given key
func WithEncryption(key []byte) func(*SecurityOptions) {
	return func(opts *SecurityOptions) {
		opts.EnableEncryption = true
		opts.EncryptionKey = key
	}
}

// WithRedaction enables redaction with the given patterns and replacement
func WithRedaction(patterns []string, replacement string) func(*SecurityOptions) {
	return func(opts *SecurityOptions) {
		opts.EnableRedaction = true
		opts.RedactionPatterns = patterns
		if replacement != "" {
			opts.RedactionReplacement = replacement
		}
	}
}

// WithIntegrityCheck enables integrity checks with the given key
func WithIntegrityCheck(key []byte) func(*SecurityOptions) {
	return func(opts *SecurityOptions) {
		opts.EnableIntegrityCheck = true
		opts.IntegrityKey = key
	}
}

// EncryptData encrypts data using AES-GCM
func EncryptData(data []byte, key []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("encryption key must be 16, 24, or 32 bytes long")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Generate a nonce (IV)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Encrypt and authenticate
	ciphertext := aesGCM.Seal(nil, nonce, data, nil)

	// Prepend nonce to ciphertext
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)

	return result, nil
}

// DecryptData decrypts data using AES-GCM
func DecryptData(data []byte, key []byte) ([]byte, error) {
	if len(data) < 12 {
		return nil, errors.New("encrypted data too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract nonce and ciphertext
	nonce := data[:12]
	ciphertext := data[12:]

	// Decrypt
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// RedactData redacts sensitive information from the given data
func RedactData(data []byte, patterns []string, replacement string) []byte {
	// Convert data to string for regex operations
	strData := string(data)

	// Apply each redaction pattern
	for _, pattern := range patterns {
		r, err := regexp.Compile(`(?i)(["']?` + pattern + `["']?\s*[:=]\s*["']?)([^"'}\s]+|[^"'}\s][^"'}\s]*[^"'}\s])`)
		if err != nil {
			// Skip invalid patterns
			continue
		}
		strData = r.ReplaceAllString(strData, "${1}"+replacement)
	}

	return []byte(strData)
}

// CalculateHMAC generates an HMAC for the given data
func CalculateHMAC(data []byte, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyHMAC checks if the HMAC for the given data matches the expected value
func VerifyHMAC(data []byte, key []byte, expectedHMAC string) bool {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	actualHMAC := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(actualHMAC), []byte(expectedHMAC))
}

// SecureEvent represents an event with security features
type SecureEvent struct {
	Event      Event  `json:"event"`       // Original event (or encrypted)
	Encrypted  bool   `json:"encrypted"`   // Whether the event is encrypted
	HMAC       string `json:"hmac"`        // HMAC for integrity verification
	IsRedacted bool   `json:"is_redacted"` // Whether the event is redacted
}

// SecureEventFromEvent creates a SecureEvent from an Event with the given security options
func SecureEventFromEvent(event Event, opts SecurityOptions) (SecureEvent, error) {
	secureEvent := SecureEvent{
		Event:      event,
		Encrypted:  false,
		IsRedacted: false,
	}

	// Convert event to JSON for processing
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return secureEvent, err
	}

	// Apply redaction if enabled
	if opts.EnableRedaction {
		redactedEventJSON := RedactData(eventJSON, opts.RedactionPatterns, opts.RedactionReplacement)
		var redactedEvent Event
		if err := json.Unmarshal(redactedEventJSON, &redactedEvent); err != nil {
			return secureEvent, err
		}
		secureEvent.Event = redactedEvent
		secureEvent.IsRedacted = true
		eventJSON = redactedEventJSON // Use redacted data for further processing
	}

	// Apply encryption if enabled
	if opts.EnableEncryption {
		encryptedData, err := EncryptData(eventJSON, opts.EncryptionKey)
		if err != nil {
			return secureEvent, err
		}
		// Replace the event with a placeholder and store encrypted data
		secureEvent.Event = Event{
			ID:        event.ID,
			Timestamp: event.Timestamp,
			Type:      event.Type,
			Details:   base64.StdEncoding.EncodeToString(encryptedData),
			File:      "", // Don't store sensitive info in plaintext
			Line:      0,
			FuncName:  "",
		}
		secureEvent.Encrypted = true
		// Update the JSON for HMAC calculation
		eventJSON, _ = json.Marshal(secureEvent.Event)
	}

	// Calculate HMAC if enabled
	if opts.EnableIntegrityCheck {
		secureEvent.HMAC = CalculateHMAC(eventJSON, opts.IntegrityKey)
	}

	return secureEvent, nil
}

// GetOriginalEvent retrieves the original event from a SecureEvent
func (se SecureEvent) GetOriginalEvent(opts SecurityOptions) (Event, error) {
	if !se.Encrypted {
		// For non-encrypted events, verify HMAC directly
		if opts.EnableIntegrityCheck && se.HMAC != "" {
			eventJSON, err := json.Marshal(se.Event)
			if err != nil {
				return Event{}, err
			}
			if !VerifyHMAC(eventJSON, opts.IntegrityKey, se.HMAC) {
				return Event{}, errors.New("HMAC verification failed: data may have been tampered with")
			}
		}
		return se.Event, nil
	}

	// Decode base64 encrypted data
	encryptedData, err := base64.StdEncoding.DecodeString(se.Event.Details)
	if err != nil {
		return Event{}, err
	}

	// For encrypted events, verify HMAC of the encrypted event
	if opts.EnableIntegrityCheck && se.HMAC != "" {
		eventJSON, err := json.Marshal(se.Event)
		if err != nil {
			return Event{}, err
		}
		if !VerifyHMAC(eventJSON, opts.IntegrityKey, se.HMAC) {
			return Event{}, errors.New("HMAC verification failed: data may have been tampered with")
		}
	}

	// Decrypt data
	decryptedData, err := DecryptData(encryptedData, opts.EncryptionKey)
	if err != nil {
		return Event{}, err
	}

	// Unmarshal decrypted data to event
	var event Event
	if err := json.Unmarshal(decryptedData, &event); err != nil {
		return Event{}, err
	}

	return event, nil
}
