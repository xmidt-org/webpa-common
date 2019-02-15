package secure

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// TokenType is a discriminator for the contents of a secure token.
type TokenType string

const (
	AuthorizationHeader string    = "Authorization"
	Invalid             TokenType = "!! INVALID !!"
	Basic               TokenType = "Basic"
	Bearer              TokenType = "Bearer"
	Digest              TokenType = "Digest"

	Untrusted = "0"
)

// ParseTokenType returns the TokenType corresponding to a string.
// This function is case-insensitive.
func ParseTokenType(value string) (TokenType, error) {
	switch {
	case strings.EqualFold(string(Basic), value):
		return Basic, nil
	case strings.EqualFold(string(Bearer), value):
		return Bearer, nil
	case strings.EqualFold(string(Digest), value):
		return Digest, nil
	default:
		return Invalid, fmt.Errorf("Invalid token type: %s", value)
	}
}

// Token is the result of parsing an authorization string
type Token struct {
	tokenType TokenType
	value     string
	trust     string
}

// String returns an on-the-wire representation of this token, suitable
// for placing into an Authorization header.
func (t *Token) String() string {
	return strings.Join(
		[]string{string(t.tokenType), t.value},
		" ",
	)
}

// Type returns the type discriminator for this token.  Note that
// the functions in this package will never create a Token with an Invalid type.
func (t *Token) Type() TokenType {
	return t.tokenType
}

func (t *Token) Value() string {
	return t.value
}

func (t *Token) Trust() string {
	return t.trust
}

func (t *Token) Bytes() []byte {
	return []byte(t.value)
}

// authorizationPattern is the regular expression that all Authorization
// strings must match to be supported by WebPA.
var authorizationPattern = regexp.MustCompile(
	fmt.Sprintf(
		`(?P<tokenType>(?i)%s|%s|%s)\s+(?P<value>.*)`,
		Basic,
		Bearer,
		Digest,
	),
)

// ParseAuthorization parses the raw Authorization string and returns a Token.
func ParseAuthorization(value string) (*Token, error) {
	matches := authorizationPattern.FindStringSubmatch(value)
	if matches == nil {
		return nil, fmt.Errorf("Invalid authorization: %s", value)
	}

	tokenType, err := ParseTokenType(matches[1])
	if err != nil {
		// There's no case where the value matches the authorizationPattern
		// will result in ParseTokenType returning an error in the current codebase.
		// This is just being very defensive ...
		return nil, err
	}

	return &Token{
		tokenType: tokenType,
		value:     matches[2],
		trust:     Untrusted,
	}, nil
}

// NewToken extracts the Authorization from the request and returns
// the Token that results from parsing that header's value.  If no
// Authorization header exists, this function returns nil with no error.
func NewToken(request *http.Request) (*Token, error) {
	value := request.Header.Get(AuthorizationHeader)
	if len(value) == 0 {
		return nil, nil
	}

	return ParseAuthorization(value)
}
