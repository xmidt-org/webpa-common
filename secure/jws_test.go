package secure

import (
	"github.com/SermoDigital/jose/jwt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestDefaultJWSParserInvalidJWS(t *testing.T) {
	assert := assert.New(t)

	token := &Token{
		tokenType: Bearer,
		value:     "",
	}

	jwsToken, err := DefaultJWSParser.ParseJWS(token)
	assert.Nil(jwsToken)
	assert.NotNil(err)
}

func TestDefaultJWSParser(t *testing.T) {
	assert := assert.New(t)

	token := &Token{
		tokenType: Bearer,
		value:     string(testSerializedJWT),
	}

	jwsToken, err := DefaultJWSParser.ParseJWS(token)
	assert.Equal(testJWT.Claims(), jwsToken.(jwt.JWT).Claims())
	assert.Nil(err)
}
