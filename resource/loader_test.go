package resource

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestLoaders(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		loader           Loader
		expectedLocation string
		expectedContents string
		expectError      bool
	}{
		{
			&fileLoader{filepath.Join(".", testFile)},
			filepath.Join(".", testFile),
			testContents,
			false,
		},
		{
			&fileLoader{filepath.Join(currentDirectory, testFile)},
			filepath.Join(currentDirectory, testFile),
			testContents,
			false,
		},
		{
			&fileLoader{"this couldn't possibly exist"},
			"this couldn't possibly exist",
			"",
			true,
		},
		{
			&urlLoader{httpServer.URL + "/" + testFile},
			httpServer.URL + "/" + testFile,
			testContents,
			false,
		},
		{
			&urlLoader{httpServer.URL + "/thisdoesnotexist"},
			httpServer.URL + "/thisdoesnotexist",
			"",
			true,
		},
		{
			&bufferLoader{[]byte("asdfasdfasdfasdfasdfasdfasdf")},
			"buffer",
			"asdfasdfasdfasdfasdfasdfasdf",
			false,
		},
		{
			&bufferLoader{[]byte{}},
			"buffer",
			"",
			false,
		},
	}

	for _, record := range testData {
		assert.Equal(record.expectedLocation, record.loader.Location())

		{
			t.Logf("verifying %#v.Open()", record.loader)
			reader, err := record.loader.Open()

			assert.Equal(reader == nil, record.expectError)
			assert.Equal(err != nil, record.expectError)

			if reader != nil {
				defer reader.Close()
				if actualContents, err := ioutil.ReadAll(reader); assert.Nil(err) {
					assert.Equal(record.expectedContents, string(actualContents))
				}
			}
		}

		{
			t.Logf("verifying ReadAll(%#v)", record.loader)
			actualContents, err := ReadAll(record.loader)

			if record.expectError {
				assert.Equal(0, len(actualContents))
				assert.NotNil(err)
			} else {
				assert.Nil(err)
				assert.Equal(record.expectedContents, string(actualContents))
			}
		}
	}
}
