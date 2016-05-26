package secure

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// TokenType is a discriminator for the contents of a secure token.
type TokenType int

const (
	Invalid TokenType = iota
	Basic
	Bearer
	Digest

	AuthorizationHeader string = "Authorization"
)

// String returns the canonical string value for a TokenType.
// This will be the prefix to an Authorization header value.
func (tt TokenType) String() string {
	switch tt {
	case Invalid:
		return "!! INVALID !!"
	case Basic:
		return "Basic"
	case Bearer:
		return "Bearer"
	case Digest:
		return "Digest"
	default:
		return "Unknown"
	}
}

// ParseTokenType returns the TokenType corresponding to a string.
// This function is case-insensitive.
func ParseTokenType(value string) (TokenType, error) {
	if strings.EqualFold(Basic.String(), value) {
		return Basic, nil
	} else if strings.EqualFold(Bearer.String(), value) {
		return Bearer, nil
	} else if strings.EqualFold(Digest.String(), value) {
		return Digest, nil
	} else {
		return Invalid, fmt.Errorf("Invalid token type: %s", value)
	}
}

// Token is the result of parsing an authorization string
type Token struct {
	tokenType TokenType
	value     string
}

// String returns an on-the-wire representation of this token, suitable
// for placing into an Authorization header.
func (t *Token) String() string {
	return fmt.Sprintf("%s %s", t.tokenType, t.value)
}

// Type returns the type discriminator for this token.  Note that
// the functions in this package will never create a Token with an Invalid type.
func (t *Token) Type() TokenType {
	return t.tokenType
}

func (t *Token) Value() string {
	return t.value
}

func (t *Token) Bytes() []byte {
	return []byte(t.value)
}

// authorizationPattern is the regular expression that all Authorization
// strings must match to be supported by WebPA.
var authorizationPattern = regexp.MustCompile(
	fmt.Sprintf(
		`(?P<tokenType>(?i)%s|%s|%s)\s+(?P<value>.*)`,
		Basic.String(),
		Bearer.String(),
		Digest.String(),
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
		return nil, err
	}

	return &Token{
		tokenType: tokenType,
		value:     matches[2],
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
