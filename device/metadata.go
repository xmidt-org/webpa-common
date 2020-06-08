package device

import (
	"sync"
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
// related to a device. Although operations are synchronized
// for both read and write, read operations are optimized as it
// is intended to store read-only values or those with infrequent
// updates. Updates are performed on a copy-on-write basis.
type Metadata struct {
	v   atomic.Value
	mux sync.Mutex
}

func (m *Metadata) loadData() map[string]interface{} {
	return m.v.Load().(map[string]interface{})
}

func (m *Metadata) storeData(data map[string]interface{}) {
	m.v.Store(data)
}

// SessionID returns the UUID associated with a device's current connection
// to the cluster.
func (m *Metadata) SessionID() string {
	sessionID, _ := m.loadData()[SessionIDKey].(string)
	return sessionID
}

// Load allows retrieving values from a device's metadata
func (m *Metadata) Load(key string) interface{} {
	return m.loadData()[key]
}

// Store allows writing values into the device's metadata given
// a key. Boolean results indicates whether the operation was successful.
// Note: operations will fail for reserved keys.
//TODO: same as SetJWTClaims()
func (m *Metadata) Store(key string, value interface{}) bool {
	if reservedMetadataKeys[key] {
		return false
	}
	m.mux.Lock()
	m.copyAndStore(key, value)
	m.mux.Unlock()
	return true
}

func (m *Metadata) copyAndStore(key string, val interface{}) {
	data := deepCopyMap(m.loadData())
	data[key] = val
	m.storeData(data)
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

// Claims returns the claims attached to a device. If no claims exist,
// they are initialized appropiately.
func (m *Metadata) Claims() map[string]interface{} {
	claims, _ := m.loadData()[JWTClaimsKey].(map[string]interface{})
	return claims
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

// SetTrustClaim sets the value of the trust in the map of claims associated with the
// device which owns this metadata.
func (m *Metadata) SetTrustClaim(trust int) {
	m.mux.Lock()
	defer m.mux.Unlock()

	claims := deepCopyMap(m.Claims())
	claims[TrustClaimKey] = trust
	m.copyAndStore(JWTClaimsKey, claims)
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
