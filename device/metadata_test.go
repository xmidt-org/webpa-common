package device

import (
	"bytes"
	"math/rand"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// claims is a convenient base claims structurally repre to use in tests.
// If you need to modify it, make a copy using deepCopyMap()
var claims map[string]interface{}

func init() {
	// This is an easy way to catch unmarshalling surprises for claims
	// which come from config values.
	rawYamlConfig := []byte(`
claims: 
  aud: XMiDT
  capabilities: [xmidt, webpa]
  custom: rbl
  exp: 1594248706
  iat: 1591656706
  id: 1234
  iss: themis
  jti: 5LnpSTsPnuh4TA
  nbf: 1591656691
  partner-id: comcast
  sub: "client:supplied"
  trust: 100
`)
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBuffer(rawYamlConfig))
	if err != nil {
		panic(err)
	}
	v.UnmarshalKey("claims", &claims)
}

func TestDeviceMetadataDefaultValues(t *testing.T) {
	assert := assert.New(t)
	m := new(Metadata)

	assert.Empty(m.Claims())
	assert.Empty(m.SessionID())
	assert.Equal(UnknownPartner, m.PartnerIDClaim())
	assert.Zero(m.TrustClaim())
	assert.Nil(m.Load("not-exists"))
}

func TestDeviceMetadataInitClaims(t *testing.T) {
	assert := assert.New(t)
	inputClaims := deepCopyMap(claims)
	m := new(Metadata)
	m.SetClaims(inputClaims)

	assert.Equal(inputClaims, m.Claims())
	assert.Equal("comcast", m.PartnerIDClaim())
	assert.Equal(100, m.TrustClaim())

	// test defensive copy
	inputClaims[TrustClaimKey] = 200
	assert.NotEqual(inputClaims, m.Claims())
}
func TestDeviceMetadataSessionID(t *testing.T) {
	assert := assert.New(t)
	m := new(Metadata)

	assert.Empty(m.SessionID())
	m.SetSessionID("uuid:123abc")
	m.SetSessionID("oopsiesCalledAgain")
	assert.Equal("uuid:123abc", m.SessionID())
}

func TestDeviceMetadataReadUpdateClaims(t *testing.T) {
	assert := assert.New(t)
	m := new(Metadata)
	m.SetClaims(claims)

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
	m := new(Metadata)
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
	m := new(Metadata)
	for reservedKey := range reservedMetadataKeys {
		before := m.Load(reservedKey)
		assert.False(m.Store(reservedKey, "poison"))
		after := m.Load(reservedKey)
		assert.Equal(before, after)
	}
}

func BenchmarkMetadataClaimsCopyParallel(b *testing.B) {
	assert := assert.New(b)
	m := new(Metadata)
	m.SetClaims(claims)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			assert.Equal(claims, m.ClaimsCopy())
		}
	})
}
func BenchmarkMetadataClaimsUsageParallel(b *testing.B) {
	b.Run("99PercentReads", benchmarkMetadataClaimsUsageParallel99PercentRead)
	b.Run("80PercentReads", benchmarkMetadataClaimsUsageParallel80PercentRead)
	b.Run("70PercentReads", benchmarkMetadataClaimsUsageParallel70PercentRead)
}

func benchmarkMetadataClaimsUsageParallel99PercentRead(b *testing.B) {
	benchmarkMetadataClaimsUsageParallel(99, b)
}

func benchmarkMetadataClaimsUsageParallel80PercentRead(b *testing.B) {
	benchmarkMetadataClaimsUsageParallel(80, b)
}

func benchmarkMetadataClaimsUsageParallel70PercentRead(b *testing.B) {
	benchmarkMetadataClaimsUsageParallel(70, b)
}

func benchmarkMetadataClaimsUsageParallel(readPercentage int, b *testing.B) {
	var mux sync.Mutex
	assert := assert.New(b)
	m := new(Metadata)
	m.SetClaims(claims)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// nolint:gosec
			v := rand.Intn(100)
			if v < readPercentage {
				m.Claims()
				m.TrustClaim()
			} else {
				mux.Lock()
				myClaimsCopy := m.ClaimsCopy()
				myTrustLevel := myClaimsCopy[TrustClaimKey].(int) + 1
				myClaimsCopy[TrustClaimKey] = myTrustLevel
				m.SetClaims(myClaimsCopy)
				assert.Equal(myTrustLevel, m.TrustClaim())
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
