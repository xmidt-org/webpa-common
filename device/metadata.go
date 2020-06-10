package device

import (
	"sync/atomic"

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
// related to a device. Read operations are optimized with a
// copy-on-write strategy. Client code must further synchronize concurrent
// writers to avoid stale data.
type Metadata struct {
	v atomic.Value
}

// SessionID returns the UUID associated with a device's current connection
// to the cluster.
func (m *Metadata) SessionID() string {
	sessionID, _ := m.loadData()[SessionIDKey].(string)
	return sessionID
}

// Load returns the value associated with the given key in the metadata map.
// It is not recommended modifying values returned by reference.
func (m *Metadata) Load(key string) interface{} {
	return m.loadData()[key]
}

// Store updates the key value mapping in the device metadata map.
// A boolean result is given indicating whether the operation was successful.
// Operations will fail for reserved keys.
// To avoid updating keys with stale data/value, client code will need to
// synchronize the entire transaction of reading, copying, modifying and
// writing back the value.
func (m *Metadata) Store(key string, value interface{}) bool {
	if reservedMetadataKeys[key] {
		return false
	}
	m.copyAndStore(key, value)
	return true
}

// NewDeviceMetadata returns a metadata object ready for use.
func NewDeviceMetadata() *Metadata {
	return NewDeviceMetadataWithClaims(make(map[string]interface{}))
}

// NewDeviceMetadataWithClaims returns a metadata object ready for use with the
// given claims.
func NewDeviceMetadataWithClaims(claims map[string]interface{}) *Metadata {
	m := new(Metadata)
	data := make(map[string]interface{})
	data[JWTClaimsKey] = deepCopyMap(claims)
	data[SessionIDKey] = ksuid.New().String()
	m.storeData(data)
	return m
}

// SetClaims updates the claims associated with the device that's
// owner of the metadata.
// To avoid updating the claims with stale data, client code will need to
// synchronize the entire transaction of reading, copying, modifying and
// writing back the value.
func (m *Metadata) SetClaims(claims map[string]interface{}) {
	m.copyAndStore(JWTClaimsKey, claims)
}

// Claims returns the claims attached to a device. The returned map
// should not be modified to avoid any race conditions. To update the claims,
// take a look at the ClaimsCopy() function
func (m *Metadata) Claims() map[string]interface{} {
	claims, _ := m.loadData()[JWTClaimsKey].(map[string]interface{})
	return claims
}

// ClaimsCopy returns a deep copy of the claims. Use this, along with the
// SetClaims() method to update the claims.
func (m *Metadata) ClaimsCopy() map[string]interface{} {
	return deepCopyMap(m.Claims())
}

// TrustClaim returns the device's trust level claim.
// By Default, a device is untrusted (trust = 0).
func (m *Metadata) TrustClaim() int {
	claims := m.Claims()
	if trust, ok := claims[TrustClaimKey].(int); ok {
		return trust
	}
	return 0
}

// PartnerIDClaim returns the partner ID claim.
// If no claim is found, the zero value is returned.
func (m *Metadata) PartnerIDClaim() string {
	claims := m.Claims()
	if partnerID, ok := claims[PartnerIDClaimKey].(string); ok {
		return partnerID
	}
	return "" // no partner by default
}

func (m *Metadata) loadData() map[string]interface{} {
	return m.v.Load().(map[string]interface{})
}

func (m *Metadata) storeData(data map[string]interface{}) {
	m.v.Store(data)
}

func (m *Metadata) copyAndStore(key string, val interface{}) {
	data := copyMap(m.loadData())
	data[key] = val
	m.storeData(data)
}

func copyMap(m map[string]interface{}) (copy map[string]interface{}) {
	copy = make(map[string]interface{})
	for k, v := range m {
		copy[k] = v
	}
	return
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
