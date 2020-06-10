package device

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// claims is a convenient base claims structurally repre to use in tests.
// If you need to modify it, make a copy using deepCopyMap()
var claims = map[string]interface{}{
	"permissions": map[string]interface{}{
		"read":  true,
		"write": false,
	},
	PartnerIDClaimKey: "comcast",
	TrustClaimKey:     100,
	"id":              1234,
	"aud":             "XMiDT",
	"custom":          "rbl",
	"exp":             1594248706,
	"iat":             1591656706,
	"iss":             "themis",
	"jti":             "5LnpSTsPnuh4TA",
	"nbf":             1591656691,
	"sub":             "client:supplied",
	"capabilities":    []string{"xmidt", "webpa"},
}

func TestDeviceMetadataInitNoClaims(t *testing.T) {
	assert := assert.New(t)
	m := NewDeviceMetadata()

	assert.Equal(claims, m.Claims())
	assert.NotEmpty(m.SessionID())
	assert.Empty(m.PartnerIDClaim())
	assert.Zero(m.TrustClaim())
	assert.Nil(m.Load("not-exists"))
}

func TestDeviceMetadataInitClaims(t *testing.T) {
	assert := assert.New(t)
	myClaims := deepCopyMap(claims)
	m := NewDeviceMetadataWithClaims(myClaims)

	assert.Equal(myClaims, m.Claims())
	assert.NotEmpty(m.SessionID())
	assert.Equal("comcast", m.PartnerIDClaim())
	assert.Equal(100, m.TrustClaim())

	// test defensive copy
	myClaims[TrustClaimKey] = 200
	assert.NotEqual(myClaims, m.Claims())
}

func TestDeviceMetadataReadUpdateClaims(t *testing.T) {
	assert := assert.New(t)
	m := NewDeviceMetadataWithClaims(claims)

	assert.Equal(claims, m.Claims())
	assert.Equal("comcast", m.PartnerIDClaim())
	assert.Equal(100, m.TrustClaim())

	myClaimsCopy := m.ClaimsCopy()
	myClaimsCopy[TrustClaimKey] = 200
	m.SetClaims(myClaimsCopy)
	assert.Equal(myClaimsCopy, m.Claims())
}

func TestDeviceMetadataUpdateCustomReferenceValue(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	m := NewDeviceMetadata()
	accountInfo := map[string]interface{}{
		"user-id":   4500,
		"user-name": "Talaria XMiDT",
	}
	require.True(m.Store("account-info", accountInfo))

	oldAccountInfo := m.Load("account-info").(map[string]interface{})
	newAccountInfo := deepCopyMap(oldAccountInfo)
	newAccountInfo["user-id"] = 4501
	require.True(m.Store("account-info", newAccountInfo))
	latestAccountInfo := m.Load("account-info").(map[string]interface{})
	assert.Equal(newAccountInfo, latestAccountInfo)
	assert.NotEqual(oldAccountInfo, latestAccountInfo)
}
func TestDeviceMetadataReservedKeys(t *testing.T) {
	assert := assert.New(t)
	m := NewDeviceMetadata()
	for reservedKey := range reservedMetadataKeys {
		before := m.Load(reservedKey)
		assert.False(m.Store(reservedKey, "poison"))
		after := m.Load(reservedKey)
		assert.Equal(before, after)
	}
}

func TestDeviceMetadataInitUserFailure(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		m := new(Metadata) // user must use the provided NewDeviceMetadata.*() functions
		m.SessionID()
	})
}

func BenchmarkMetadataClaimsCopyParallel(b *testing.B) {
	assert := assert.New(b)
	m := NewDeviceMetadataWithClaims(claims)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			assert.Equal(claims, m.ClaimsCopy())
		}
	})
}
func BenchmarkMetadataClaimsUsageParallel(b *testing.B) {
	var mux sync.Mutex
	m := NewDeviceMetadataWithClaims(claims)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v := rand.Intn(100)
			if v < 95 { // Perform only reads 95% of the time.
				m.Claims()
				m.TrustClaim()
			} else {
				mux.Lock()
				myClaimsCopy := m.ClaimsCopy()
				myClaimsCopy[TrustClaimKey] = myClaimsCopy[TrustClaimKey].(int) + 1
				m.SetClaims(myClaimsCopy)
				mux.Unlock()
			}
		}
	})
}

func TestDeepCopyMap(t *testing.T) {
	testCases := []struct {
		Name     string
		Input    map[string]interface{}
		Expected map[string]interface{}
	}{
		{
			Name:     "Nil",
			Expected: make(map[string]interface{}),
		},
		{
			Name:     "Empty",
			Input:    make(map[string]interface{}),
			Expected: make(map[string]interface{}),
		},
		{
			Name:     "Simple",
			Input:    map[string]interface{}{"k0": 0, "k1": 1},
			Expected: map[string]interface{}{"k0": 0, "k1": 1},
		},
		{
			Name: "Complex",
			Input: map[string]interface{}{
				"nested": map[string]interface{}{
					"nestedKey": "nestedVal",
				},
				"nestedToCast": map[interface{}]interface{}{
					3: "nestedVal3",
				},
			},
			Expected: map[string]interface{}{
				"nested": map[string]interface{}{
					"nestedKey": "nestedVal",
				},
				"nestedToCast": map[string]interface{}{
					"3": "nestedVal3",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			cpy := deepCopyMap(testCase.Input)
			assert.Equal(cpy, testCase.Expected)
		})
	}
}
