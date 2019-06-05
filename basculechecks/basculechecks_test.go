package basculechecks

import (
	"context"
	"errors"
	"testing"

	"github.com/Comcast/comcast-bascule/bascule"
	"github.com/stretchr/testify/assert"
)

func TestValidCapabilityCheck(t *testing.T) {
	config := CapabilityConfig{
		FirstPiece:      "a",
		SecondPiece:     "b",
		ThirdPiece:      "c",
		AcceptAllMethod: "all",
	}
	check := CreateValidCapabilityCheck(config)
	goodAuth := bascule.Authentication{
		Authorization: "jwt",
		Token:         bascule.NewToken("Bearer", "jwt", bascule.Attributes{}),
		Request: bascule.Request{
			URL:    "/something/test",
			Method: "GET",
		},
	}
	goodContext := bascule.WithAuthentication(context.Background(), goodAuth)
	goodVals := []interface{}{
		"d:e:f:/aaaa:post",
		"a:b:d:/aaaa:all",
		`a:b:c:test\b:get`,
	}
	tests := []struct {
		description string
		ctx         context.Context
		vals        []interface{}
		expectedErr error
	}{
		{
			description: "Success",
			ctx:         goodContext,
			vals:        goodVals,
		},
		{
			description: "No Vals Error",
			expectedErr: ErrNoVals,
		},
		{
			description: "No Auth Error",
			ctx:         context.Background(),
			vals:        goodVals,
			expectedErr: ErrNoAuth,
		},
		{
			description: "Nonstring Val Error",
			ctx:         goodContext,
			vals:        []interface{}{3},
			expectedErr: ErrNonstringVal,
		},
		{
			description: "Empty String Error",
			ctx:         goodContext,
			vals:        []interface{}{""},
			expectedErr: ErrEmptyString,
		},
		{
			description: "Malformed String Error",
			ctx:         goodContext,
			vals:        []interface{}{"::"},
			expectedErr: errors.New("malformed string"),
		},
		{
			description: "No Valid Capability Error",
			ctx:         goodContext,
			vals:        []interface{}{"::::"},
			expectedErr: ErrNoValidCapabilityFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := check(tc.ctx, tc.vals)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}
