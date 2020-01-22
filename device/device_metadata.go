package device

import (
	"encoding/json"
	"github.com/segmentio/ksuid"
)

const (
	// JWTClaimsKey is a reserved metadata key
	JWTClaimsKey = "jwt-claims"

	// Top level JWT Claim keys
	PartnerIDClaimKey = "partner-id"
	TrustClaimKey     = "trust"

	// Top level Session ID
	SessionKey = "session-id"
)

// Metadata contains information such as security credentials
// related to a device
type Metadata map[string]interface{}

// JWTClaims returns the JWT claims attached to a device
func (m Metadata) JWTClaims() *JWTClaims { // returns the type and such type has getter/setter
	if jwtClaims, ok := m[JWTClaimsKey].(JWTClaims); ok {
		return &jwtClaims
	}
	return nil
}

func (m Metadata) SetJWTClaims(jwtClaims JWTClaims) {
	m[JWTClaimsKey] = jwtClaims
}

// Load allows retrieving values from a device's metadata
func (m Metadata) Load(key string) interface{} {
	return m[key]
}

// Store allows writing values into the device's metadata given
// a key.
// Note: 'jwt-claims' is a reserved key so store calls will fail
// if that key is used.
func (m Metadata) Store(key string, value interface{}) bool {
	if key == JWTClaimsKey || key == SessionKey {
		return false
	}
	m[key] = value
	return true
}

// NewDeviceMetadata returns a metadata object ready for use
func NewDeviceMetadata() Metadata {
	m := make(Metadata)
	m.SetJWTClaims(NewJWTClaims(make(map[string]interface{})))
	m[SessionKey] = ksuid.New()
	return m
}

// NewJWTClaims is a convenience constructor useful for setting
// claims on an existing device metadata object
func NewJWTClaims(claims map[string]interface{}) JWTClaims {
	return JWTClaims{
		data: claims,
	}
}

// JWTClaims defines the allowed interactions with the claims
// in a device's metadata
type JWTClaims struct {
	data map[string]interface{}
}

// Data returns the internal claims map
func (c *JWTClaims) Data() map[string]interface{} {
	return c.data // TODO: return a deep copy if security/mutability is a concern?
}

// Trust returns the device's trust level claim
// By Default, a device is untrusted (trust = 0).
func (c *JWTClaims) Trust() int {
	if trust, ok := c.data[TrustClaimKey].(int); ok {
		return trust
	}
	return 0
}

// PartnerIDs returns a singleton list with the partnerID
// the device presented during registration
// By default, an empty list is returned.
func (c *JWTClaims) PartnerIDs() []string {
	if partnerID, ok := c.data[PartnerIDClaimKey].(string); ok {
		return []string{partnerID}
	}
	return []string{} // no partners by default
}

func (c *JWTClaims) SetTrust(trust int) {
	c.data[TrustClaimKey] = trust
}

func (c *JWTClaims) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.data)
}
