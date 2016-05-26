package secure

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestParseTokenType(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		value    string
		expected TokenType
	}{
		{"Basic", Basic},
		{"BASIC", Basic},
		{"BasIC", Basic},
		{"Bearer", Bearer},
		{"bearer", Bearer},
		{"bEARer", Bearer},
		{"Digest", Digest},
		{"DIGEst", Digest},
		{"DigeSt", Digest},
		{"asdfasdf", Invalid},
		{"", Invalid},
		{"   ", Invalid},
	}

	for _, record := range testData {
		tokenType, err := ParseTokenType(record.value)
		assert.Equal(record.expected, tokenType)
		assert.Equal(err == nil, tokenType != Invalid)
	}
}

func TestValidTokens(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		token        Token
		expectString string
		expectType   TokenType
		expectValue  string
	}{
		{
			Token{Basic, "dXNlcjpwYXNzd29yZA=="},
			"Basic dXNlcjpwYXNzd29yZA==",
			Basic,
			"dXNlcjpwYXNzd29yZA==",
		},
		{
			Token{Bearer, "eyJraWQiOiJzYXQtc3RnLWsxLTEwMjQiLCJhbGciOiJSUzI1NiJ9.eyJqdGkiOiI4ZDIwYzY4Zi04NjM5LTQyNDEtYTY2Yi00OTllYmFlYTQ0ZDMiLCJpc3MiOiJzYXRzLXN0YWdpbmciLCJzdWIiOiJ4MTpzdGc6bXNvOmNpc2NvOnNlcnZpY2VncmlkOmNveDoxN2ZkNTciLCJpYXQiOjE0NTgzMDg1OTEsIm5iZiI6MTQ1ODMwODU5MSwiZXhwIjoxNDU4Mzk0OTk0LCJ2ZXJzaW9uIjoiMS4wIiwiYWxsb3dlZFJlc291cmNlcyI6eyJhbGxvd2VkUGFydG5lcnMiOlsiY294Il19LCJjYXBhYmlsaXRpZXMiOltdLCJhdWQiOltdfQ.ieHnWWjO-CbvUJ_x9RJaMpOdAKqad0b8Rdd322dlxulqJud_O3fbSYqcSX3Sl1X8KySqgr7sHvBJAET43c_Agumj8d8vK3eCaCV-8d2W3SkBB4ePB4b1D6qpg02kqF5eXst1CdMUixYa0fw1PBvCHZe2s_M-qjW7qv5DF73wgsg"},
			"Bearer eyJraWQiOiJzYXQtc3RnLWsxLTEwMjQiLCJhbGciOiJSUzI1NiJ9.eyJqdGkiOiI4ZDIwYzY4Zi04NjM5LTQyNDEtYTY2Yi00OTllYmFlYTQ0ZDMiLCJpc3MiOiJzYXRzLXN0YWdpbmciLCJzdWIiOiJ4MTpzdGc6bXNvOmNpc2NvOnNlcnZpY2VncmlkOmNveDoxN2ZkNTciLCJpYXQiOjE0NTgzMDg1OTEsIm5iZiI6MTQ1ODMwODU5MSwiZXhwIjoxNDU4Mzk0OTk0LCJ2ZXJzaW9uIjoiMS4wIiwiYWxsb3dlZFJlc291cmNlcyI6eyJhbGxvd2VkUGFydG5lcnMiOlsiY294Il19LCJjYXBhYmlsaXRpZXMiOltdLCJhdWQiOltdfQ.ieHnWWjO-CbvUJ_x9RJaMpOdAKqad0b8Rdd322dlxulqJud_O3fbSYqcSX3Sl1X8KySqgr7sHvBJAET43c_Agumj8d8vK3eCaCV-8d2W3SkBB4ePB4b1D6qpg02kqF5eXst1CdMUixYa0fw1PBvCHZe2s_M-qjW7qv5DF73wgsg",
			Bearer,
			"eyJraWQiOiJzYXQtc3RnLWsxLTEwMjQiLCJhbGciOiJSUzI1NiJ9.eyJqdGkiOiI4ZDIwYzY4Zi04NjM5LTQyNDEtYTY2Yi00OTllYmFlYTQ0ZDMiLCJpc3MiOiJzYXRzLXN0YWdpbmciLCJzdWIiOiJ4MTpzdGc6bXNvOmNpc2NvOnNlcnZpY2VncmlkOmNveDoxN2ZkNTciLCJpYXQiOjE0NTgzMDg1OTEsIm5iZiI6MTQ1ODMwODU5MSwiZXhwIjoxNDU4Mzk0OTk0LCJ2ZXJzaW9uIjoiMS4wIiwiYWxsb3dlZFJlc291cmNlcyI6eyJhbGxvd2VkUGFydG5lcnMiOlsiY294Il19LCJjYXBhYmlsaXRpZXMiOltdLCJhdWQiOltdfQ.ieHnWWjO-CbvUJ_x9RJaMpOdAKqad0b8Rdd322dlxulqJud_O3fbSYqcSX3Sl1X8KySqgr7sHvBJAET43c_Agumj8d8vK3eCaCV-8d2W3SkBB4ePB4b1D6qpg02kqF5eXst1CdMUixYa0fw1PBvCHZe2s_M-qjW7qv5DF73wgsg",
		},
	}

	for _, record := range testData {
		assert.Equal(record.token.String(), record.expectString)
		assert.Equal(record.token.Type(), record.expectType)
		assert.Equal(record.token.Value(), record.expectValue)
		assert.Equal(record.token.Bytes(), []byte(record.expectValue))

		if parsedToken, err := ParseAuthorization(record.expectString); assert.Nil(err) {
			assert.Equal(record.token, *parsedToken)
		}

		if request, err := http.NewRequest("GET", "", nil); assert.Nil(err) {
			request.Header.Add(AuthorizationHeader, record.expectString)
			if fromRequest, err := NewToken(request); assert.Nil(err) {
				assert.Equal(record.token, *fromRequest)
			}
		}
	}
}

func TestInvalidTokens(t *testing.T) {
	assert := assert.New(t)
	var invalidTokens = []string{
		"SomePRefix dXNlcjpwYXNzd29yZA==",
		"dXNlcjpwYXNzd29yZA==",
		"eyJraWQiOiJzYXQtc3RnLWsxLTEwMjQiLCJhbGciOiJSUzI1NiJ9.eyJqdGkiOiI4ZDIwYzY4Zi04NjM5LTQyNDEtYTY2Yi00OTllYmFlYTQ0ZDMiLCJpc3MiOiJzYXRzLXN0YWdpbmciLCJzdWIiOiJ4MTpzdGc6bXNvOmNpc2NvOnNlcnZpY2VncmlkOmNveDoxN2ZkNTciLCJpYXQiOjE0NTgzMDg1OTEsIm5iZiI6MTQ1ODMwODU5MSwiZXhwIjoxNDU4Mzk0OTk0LCJ2ZXJzaW9uIjoiMS4wIiwiYWxsb3dlZFJlc291cmNlcyI6eyJhbGxvd2VkUGFydG5lcnMiOlsiY294Il19LCJjYXBhYmlsaXRpZXMiOltdLCJhdWQiOltdfQ.ieHnWWjO-CbvUJ_x9RJaMpOdAKqad0b8Rdd322dlxulqJud_O3fbSYqcSX3Sl1X8KySqgr7sHvBJAET43c_Agumj8d8vK3eCaCV-8d2W3SkBB4ePB4b1D6qpg02kqF5eXst1CdMUixYa0fw1PBvCHZe2s_M-qjW7qv5DF73wgsg",
	}

	for _, invalidToken := range invalidTokens {
		t.Logf("Testing invalid token: %s", invalidToken)
		parsedToken, err := ParseAuthorization(invalidToken)
		assert.Nil(parsedToken)
		assert.NotNil(err)

		if request, err := http.NewRequest("GET", "", nil); assert.Nil(err) {
			request.Header.Add(AuthorizationHeader, invalidToken)
			fromRequest, err := NewToken(request)
			assert.Nil(fromRequest)
			assert.NotNil(err)
		}
	}
}

func TestNoAuthorization(t *testing.T) {
	assert := assert.New(t)
	if request, err := http.NewRequest("GET", "", nil); assert.Nil(err) {
		fromRequest, err := NewToken(request)
		assert.Nil(fromRequest)
		assert.Nil(err)
	}
}
