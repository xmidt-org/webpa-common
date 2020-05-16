package device

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceMetadata(t *testing.T) {
	for _, mode := range []bool{true, false} {
		strMode := strconv.FormatBool(mode)
		t.Run("Init/bare/"+strMode, testDeviceMetadataInit(mode))
		t.Run("Update/bare/"+strMode, testDeviceMetadataUpdate(mode))
		t.Run("ProtectedKeys/bare/"+strMode, testDeviceMetadataProtectedKeys(mode))
	}
}

func testDeviceMetadataProtectedKeys(bare bool) func(t *testing.T) {
	return func(t *testing.T) {
		var (
			m      Metadata
			assert = assert.New(t)
			claims = map[string]interface{}{
				"claim0":          "v0",
				"claim1":          1,
				TrustClaimKey:     88,
				PartnerIDClaimKey: "comcast",
			}
		)

		if bare {
			m = make(Metadata)
			m.SetJWTClaims(JWTClaims(claims))
		} else {
			m = NewDeviceMetadataWithClaims(claims)
		}
		sid := m.SessionID()
		assert.False(m.Store(SessionIDKey, "oops, protected key"))
		assert.Equal(sid, m.SessionID())
	}

}
func testDeviceMetadataUpdate(bare bool) func(*testing.T) {
	return func(t *testing.T) {
		var (
			m              Metadata
			assert         = assert.New(t)
			require        = require.New(t)
			providedClaims = map[string]interface{}{
				TrustClaimKey:     88,
				PartnerIDClaimKey: "comcast",
			}

			expectedInitialClaims = JWTClaims(map[string]interface{}{
				TrustClaimKey:     88,
				PartnerIDClaimKey: "comcast",
			})
		)

		if bare {
			m = make(Metadata)
			m.SetJWTClaims(providedClaims)
		} else {
			m = NewDeviceMetadataWithClaims(providedClaims)
		}

		require.NotNil(m)
		jwtClaims := m.JWTClaims()
		require.NotNil(jwtClaims)
		assert.Equal(expectedInitialClaims, jwtClaims)
		assert.Equal(providedClaims[PartnerIDClaimKey], jwtClaims.PartnerID())
		assert.Equal(providedClaims[TrustClaimKey], jwtClaims.Trust())

		jwtClaims.SetTrust(100)
		assert.Equal(100, jwtClaims.Trust())
		assert.NotEqual(providedClaims[TrustClaimKey], jwtClaims.Trust())
		assert.NotEqual(jwtClaims, m.JWTClaims())
	}
}

func testDeviceMetadataInit(bare bool) func(t *testing.T) {
	return func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)
		var m Metadata

		if bare {
			m = make(Metadata)
		} else {
			m = NewDeviceMetadata()
		}

		assert.NotEmpty(m.SessionID())

		claims := m.JWTClaims()
		require.NotNil(claims)
		assert.Empty(claims)
		assert.Empty(claims.PartnerID())
		assert.Zero(claims.Trust())

		input := "myValue"
		assert.True(m.Store("k", input))
		output := m.Load("k")
		assert.Equal(input, output)
	}
}

func BenchmarkMetadataJWTClaimsAccessParallel(b *testing.B) {
	m := NewDeviceMetadataWithClaims(map[string]interface{}{
		"k0": "v0",
		"k1": "v1",
	})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jwtClaims := m.JWTClaims()
			for k, v := range jwtClaims {
				jwtClaims[k+"prime"] = v
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
