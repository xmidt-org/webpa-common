package basculechecks

// func TestValidCapabilityCheckFail(t *testing.T) {
// 	check, err := CreateValidCapabilityCheck(`\K`, "")
// 	assert.NotNil(t, err)
// 	assert.Nil(t, check)
// }

// func TestValidCapabilityCheck(t *testing.T) {
// 	acceptAll := "all"
// 	prefix := "a:b:c:"
// 	check, err := CreateValidCapabilityCheck(prefix, acceptAll)
// 	assert.Nil(t, err)
// 	u, err := url.Parse("/test")
// 	assert.Nil(t, err)
// 	goodAuth := bascule.Authentication{
// 		Authorization: "jwt",
// 		Token:         bascule.NewToken("Bearer", "jwt", bascule.Attributes{}),
// 		Request: bascule.Request{
// 			URL:    u,
// 			Method: "GET",
// 		},
// 	}
// 	goodContext := bascule.WithAuthentication(context.Background(), goodAuth)
// 	goodVals := []interface{}{
// 		"d:e:f:/aaaa:post",
// 		"a:b:d:/aaaa:all",
// 		`a:b:c:/test\b:post`,
// 		`a:b:c:z:all`,
// 		`a:b:c:/test\b:get`,
// 	}
// 	tests := []struct {
// 		description string
// 		ctx         context.Context
// 		vals        []interface{}
// 		expectedErr error
// 	}{
// 		{
// 			description: "Success",
// 			ctx:         goodContext,
// 			vals:        goodVals,
// 		},
// 		{
// 			description: "No Vals Error",
// 			expectedErr: ErrNoVals,
// 		},
// 		{
// 			description: "No Auth Error",
// 			ctx:         context.Background(),
// 			vals:        goodVals,
// 			expectedErr: ErrNoAuth,
// 		},
// 		{
// 			description: "Nonstring Val Error",
// 			ctx:         goodContext,
// 			vals:        []interface{}{3},
// 			expectedErr: ErrNonstringVal,
// 		},
// 		{
// 			description: "No Valid Capability Error",
// 			ctx:         goodContext,
// 			vals:        []interface{}{"::::"},
// 			expectedErr: ErrNoValidCapabilityFound,
// 		},
// 	}

// 	for _, tc := range tests {
// 		t.Run(tc.description, func(t *testing.T) {
// 			assert := assert.New(t)
// 			err := check(tc.ctx, tc.vals)
// 			if tc.expectedErr == nil || err == nil {
// 				assert.Equal(tc.expectedErr, err)
// 			} else {
// 				assert.Contains(err.Error(), tc.expectedErr.Error())
// 			}
// 		})
// 	}
// }
