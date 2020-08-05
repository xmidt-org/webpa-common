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
// related to a device. Read operations are optimized with a
// copy-on-write strategy. Client code must further synchronize concurrent
// writers to avoid stale data.
// Metadata uses an atomic.Value internally and thus it should not be copied
// after creation.
type Metadata struct {
	v    atomic.Value
	once sync.Once
}

// SessionID returns the UUID associated with a device's current connection
// to the cluster if one has been set. The zero value is returned as default.
func (m *Metadata) SessionID() (sessionID string) {
	sessionID, _ = m.loadData()[SessionIDKey].(string)
	return
}

// SetSessionID sets the UUID associated the device's current connection to the cluster.
// It uses sync.Once to ensure the sessionID is unchanged through the metadata's lifecycle.
func (m *Metadata) SetSessionID(sessionID string) {
	m.once.Do(func() {
		m.copyAndStore(SessionIDKey, sessionID)
	})
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

// SetClaims updates the claims associated with the device that's
// owner of the metadata.
// To avoid updating the claims with stale data, client code will need to
// synchronize the entire transaction of reading, copying, modifying and
// writing back the value.
func (m *Metadata) SetClaims(claims map[string]interface{}) {
	m.copyAndStore(JWTClaimsKey, deepCopyMap(claims))
}

// Claims returns the claims attached to a device. The returned map
// should not be modified to avoid any race conditions. To update the claims,
// take a look at the ClaimsCopy() function
func (m *Metadata) Claims() (claims map[string]interface{}) {
	claims, _ = m.loadData()[JWTClaimsKey].(map[string]interface{})
	return
}

// ClaimsCopy returns a deep copy of the claims. Use this, along with the
// SetClaims() method to update the claims.
func (m *Metadata) ClaimsCopy() map[string]interface{} {
	return deepCopyMap(m.Claims())
}

// TrustClaim returns the device's trust level claim.
// By Default, a device is untrusted (trust = 0).
func (m *Metadata) TrustClaim() int {
	return cast.ToInt(m.Claims()[TrustClaimKey])
}

// PartnerIDClaim returns the partner ID claim.
// If no claim is found, the zero value is returned.
func (m *Metadata) PartnerIDClaim() (partnerID string) {
	partnerID, _ = m.Claims()[PartnerIDClaimKey].(string)
	return
}

func (m *Metadata) loadData() (data map[string]interface{}) {
	data, _ = m.v.Load().(map[string]interface{})
	return
}

func (m *Metadata) storeData(data map[string]interface{}) {
	m.v.Store(data)
}

func (m *Metadata) copyAndStore(key string, val interface{}) {
	data := deepCopyMap(m.loadData())
	data[key] = val
	m.storeData(data)
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
