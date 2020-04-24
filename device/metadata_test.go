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
			m       Metadata
			assert  = assert.New(t)
			require = require.New(t)
			claims  = map[string]interface{}{
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

		require.NotNil(m)
		jwtClaims := m.JWTClaims()
		require.NotNil(jwtClaims)
		assert.Equal(claims, jwtClaims.ToMap())
		assert.Equal("comcast", jwtClaims.PartnerID())
		assert.Equal(88, jwtClaims.Trust())

		jwtClaims.SetTrust(100)
		assert.Equal(100, jwtClaims.Trust())
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

		claims := m.JWTClaims()

		require.NotNil(claims)
		assert.NotEmpty(m.SessionID())
		assert.Empty(claims.PartnerID())
		assert.Zero(claims.Trust())

		input := "myValue"
		assert.True(m.Store("k", input))
		output := m.Load("k")
		assert.Equal(input, output)
	}
}
