package device

import (
	"encoding/json"
	"sync"

	"github.com/segmentio/ksuid"
	"github.com/spf13/cast"
)

// Reserved metadata keys.
const (
	JWTClaimsKey = "jwt-claims"
	SessionIDKey = "session-id"
)

// Top level JWTClaim keys.
const (
	PartnerIDClaimKey = "partner-id"
	TrustClaimKey     = "trust"
)

// Default values for well known claims.
const (
	DefaultPartnerID  = ""
	DefaultTrustLevel = 0
)

var reservedMetadataKeys = map[string]bool{
	JWTClaimsKey: true, SessionIDKey: true,
}

func init() {
	ksuid.SetRand(ksuid.FastRander)
}

// Metadata contains information such as security credentials
// related to a device.
type Metadata struct {
	mux  sync.RWMutex
	data map[string]interface{}
}

// JWTClaims returns a read-only view of the JWT claims attached to a device.
func (m Metadata) JWTClaims() map[string]interface{} {
	m.mux.RLock()
	defer m.mux.RUnlock()

	jwtClaims, _ := m.data[JWTClaimsKey].(map[string]interface{})
	return deepCopyMap(jwtClaims)
}

// SetJWTClaims sets the JWT claims attached to a device. If known
// claims are not provided, default values are injected.
func (m Metadata) SetJWTClaims(claims map[string]interface{}) {
	claims = deepCopyMap(claims)
	_, ok := claims[TrustClaimKey]
	if !ok {
		claims[TrustClaimKey] = 0
	}

	_, ok = claims[PartnerIDClaimKey]
	if !ok {
		claims[PartnerIDClaimKey] = ""

	}

	m.mux.Lock()
	m.data[JWTClaimsKey] = claims
	m.mux.Unlock()
}

// SessionID returns the UUID associated with a device's current connection
// to the cluster.
func (m Metadata) SessionID() string {
	if sessionID, ok := m.data[SessionIDKey].(string); ok {
		return sessionID
	}

	return m.initSessionID()
}

func (m Metadata) initSessionID() string {
	sessionID := ksuid.New().String()
	m[SessionIDKey] = sessionID
	return sessionID
}

// Load allows retrieving values from a device's metadata
func (m Metadata) Load(key string) interface{} {
	return m[key]
}

// Store allows writing values into the device's metadata given
// a key. Boolean results indicates whether the operation was successful.
// Note: operations will fail for reserved keys.
func (m Metadata) Store(key string, value interface{}) bool {
	if reservedMetadataKeys[key] {
		return false
	}
	m[key] = value
	return true
}

// NewDeviceMetadata returns a metadata object ready for use.
func NewDeviceMetadata() Metadata {
	return NewDeviceMetadataWithClaims(make(map[string]interface{}))
}

// NewDeviceMetadataWithClaims returns a metadata object ready for use with the
// given claims.
func NewDeviceMetadataWithClaims(claims map[string]interface{}) Metadata {
	m := Metadata{
		data: make(map[string]interface{}),
	}
	m.SetJWTClaims(claims)
	m.SetJWTClaims(deepCopyMap(claims))
	m.initSessionID()
	return m
}

// Trust returns the device's trust level claim
// By Default, a device is untrusted (trust = 0).
func (c JWTClaims) Trust() int {
	if trust, ok := c[TrustClaimKey].(int); ok {
		return trust
	}
	return 0
}

// PartnerID returns the partner ID claim.
// If no claim is found, the zero value is returned.
func (c JWTClaims) PartnerID() string {
	if partnerID, ok := c[PartnerIDClaimKey].(string); ok {
		return partnerID
	}
	return "" // no partner by default
}

// SetTrust modifies the trust level of the device which owns these
// claims.
func (c JWTClaims) SetTrust(trust int) {
	c[TrustClaimKey] = trust
}

// MarshalJSON allows easy JSON representation of the JWTClaims underlying claims map.
func (c JWTClaims) MarshalJSON() ([]byte, error) {
	return json.Marshal(c)
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	deepCopy := make(map[string]interface{})
	for key, val := range m {
		switch val.(type) {
		case map[interface{}]interface{}:
			val = cast.ToStringMap(val)
			deepCopy[key] = deepCopyMap(val.(map[string]interface{}))
		case map[string]interface{}:
			deepCopy[key] = deepCopyMap(val.(map[string]interface{}))
		default:
			deepCopy[key] = val
		}

	}
	return deepCopy
}
