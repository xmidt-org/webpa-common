package secure

import (
	"github.com/SermoDigital/jose/jws"
)

// JWSParser parses raw Tokens into JWS objects
type JWSParser interface {
	ParseJWS(*Token) (jws.JWS, error)
}

type defaultJWSParser int

func (parser defaultJWSParser) ParseJWS(token *Token) (jws.JWS, error) {
	if jwtToken, err := jws.ParseJWT(token.Bytes()); err == nil {
		if trust, ok := jwtToken.Claims().Get("trust").(string); ok {
			if len(trust) > 0 {
				token.trust = trust
			}
		}

		return jwtToken.(jws.JWS), nil
	} else {
		return nil, err
	}
}

// DefaultJWSParser is the parser implementation that simply delegates to
// the SermoDigital library's jws.ParseJWT function.
var DefaultJWSParser JWSParser = defaultJWSParser(0)
