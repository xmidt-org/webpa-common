package resource

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestLocationIO(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		location       Location
		expectContents string
		expectError    bool
	}{
		{
			Location(filepath.Join(".", testFile)),
			testContents,
			false,
		},
		{
			Location(filepath.Join(currentDirectory, testFile)),
			testContents,
			false,
		},
		{
			Location("this couldn't possibly exist"),
			"",
			true,
		},
		{
			Location(httpServer.URL + "/" + testFile),
			testContents,
			false,
		},
		{
			Location(httpServer.URL + "/thisdoesnotexist.txt"),
			"",
			true,
		},
	}

	for _, record := range testData {
		{
			t.Logf("Opening %s", record.location)
			reader, err := record.location.Open()
			if reader != nil {
				defer reader.Close()
			}

			t.Logf("reader: %v, err: %v", reader, err)
			assert.Equal(reader == nil, record.expectError)
			assert.Equal(err != nil, record.expectError)

			if !record.expectError {
				actualContents, err := ioutil.ReadAll(reader)
				if assert.Nil(err) {
					assert.Equal(record.expectContents, string(actualContents))
				}
			}
		}

		{
			t.Logf("Reading %s", record.location)
			actualContents, err := record.location.ReadAll()

			t.Logf("actualContents: %s, err: %v", actualContents, err)
			assert.Equal(actualContents == nil, record.expectError)
			assert.Equal(err != nil, record.expectError)

			if !record.expectError {
				assert.Equal(record.expectContents, string(actualContents))
			}
		}
	}
}
