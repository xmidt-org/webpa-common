package device

import (
	"encoding/json"

	"github.com/segmentio/ksuid"
	"github.com/spf13/cast"
)

//Reserved metadata keys
const (
	JWTClaimsKey = "jwt-claims"
	SessionIDKey = "session-id"
)

//Top level JWTClaim keys
const (
	PartnerIDClaimKey = "partner-id"
	TrustClaimKey     = "trust"
)

var reservedMetadataKeys = map[string]bool{
	JWTClaimsKey: true, SessionIDKey: true,
}

func init() {
	ksuid.SetRand(ksuid.FastRander)
}

// Metadata contains information such as security credentials
// related to a device.
type Metadata map[string]interface{}

// JWTClaims returns the JWT claims attached to a device. If no claims exist,
// they are initialized appropiately.
func (m Metadata) JWTClaims() JWTClaims { // returns the type and such type has getter/setter
	if jwtClaims, ok := m[JWTClaimsKey].(JWTClaims); ok {
		return deepCopyMap(jwtClaims)
	}
	return deepCopyMap(m.initJWTClaims())
}

// SetJWTClaims sets the JWT claims attached to a device.
func (m Metadata) SetJWTClaims(jwtClaims JWTClaims) {
	m[JWTClaimsKey] = jwtClaims
}

// SessionID returns the UUID associated with a device's current connection
// to the cluster.
func (m Metadata) SessionID() string {
	if sessionID, ok := m[SessionIDKey].(string); ok {
		return sessionID
	}

	return m.initSessionID()
}

func (m Metadata) initSessionID() string {
	sessionID := ksuid.New().String()
	m[SessionIDKey] = sessionID
	return sessionID
}

func (m Metadata) initJWTClaims() JWTClaims {
	jwtClaims := JWTClaims(make(map[string]interface{}))
	m.SetJWTClaims(jwtClaims)
	return jwtClaims
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
	m := make(Metadata)
	m.SetJWTClaims(deepCopyMap(claims))
	m.initSessionID()
	return m
}

// JWTClaims defines the interface of a device's security claims.
// One current use case is providing security credentials the device
// presented at registration time.
type JWTClaims map[string]interface{}

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
