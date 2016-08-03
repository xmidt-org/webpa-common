package resource

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFactoryAmbiguousResource(t *testing.T) {
	assert := assert.New(t)

	factory := &Factory{
		URI:  "http://someplace.com/foobar.txt",
		Data: "here is some content",
	}

	url, err := factory.URL()
	assert.Nil(url)
	assert.Equal(ErrorAmbiguousResource, err)

	loader, err := factory.NewLoader()
	assert.Nil(loader)
	assert.Equal(ErrorAmbiguousResource, err)

	template, err := factory.NewExpander()
	assert.Nil(template)
	assert.Equal(ErrorAmbiguousResource, err)
}

func TestFactoryNoResource(t *testing.T) {
	assert := assert.New(t)

	factory := &Factory{}

	url, err := factory.URL()
	assert.Nil(url)
	assert.Equal(ErrorNoResource, err)

	loader, err := factory.NewLoader()
	assert.Nil(loader)
	assert.Equal(ErrorNoResource, err)

	template, err := factory.NewExpander()
	assert.Nil(template)
	assert.Equal(ErrorURIRequired, err)
}

func TestFactoryUnsupportedScheme(t *testing.T) {
	assert := assert.New(t)

	factory := &Factory{
		URI: "whatisthis://foo/bar.txt",
	}

	url, err := factory.URL()
	assert.Nil(url)
	assert.NotNil(err)

	loader, err := factory.NewLoader()
	assert.Nil(loader)
	assert.NotNil(err)
}

func TestFactoryBadURI(t *testing.T) {
	assert := assert.New(t)

	factory := &Factory{
		URI: "http://        /what/\t",
	}

	url, err := factory.URL()
	assert.Nil(url)
	assert.NotNil(err)

	loader, err := factory.NewLoader()
	assert.Nil(loader)
	assert.NotNil(err)
}

func TestFactoryBadTemplate(t *testing.T) {
	assert := assert.New(t)

	factory := &Factory{
		URI: "http://example.com/{bad",
	}

	url, err := factory.URL()
	assert.NotNil(url)
	assert.Nil(err)

	template, err := factory.NewExpander()
	assert.Nil(template)
	assert.NotNil(err)
}

func TestFactoryData(t *testing.T) {
	assert := assert.New(t)

	message := "here is some lovely content"
	factory := &Factory{
		Data: message,
	}

	url, err := factory.URL()
	assert.Nil(url)
	assert.Nil(err)

	if loader, err := factory.NewLoader(); assert.NotNil(loader) && assert.Nil(err) {
		assert.Equal(message, loader.Location())
		data, err := ReadAll(loader)
		assert.Equal(message, string(data))
		assert.Nil(err)
	}

	template, err := factory.NewExpander()
	assert.Nil(template)
	assert.Equal(ErrorURIRequired, err)
}

func TestFactoryFileLoader(t *testing.T) {
	assert := assert.New(t)

	for _, fileURI := range []string{testFilePath, testFileURI} {
		t.Logf("fileURI: %s", fileURI)
		factory := &Factory{
			URI: fileURI,
		}

		if url, err := factory.URL(); assert.NotNil(url) && assert.Nil(err) {
			assert.Equal(FileScheme, url.Scheme)

			if loader, err := factory.NewLoader(); assert.NotNil(loader) && assert.Nil(err) {
				assert.Equal(url.Path, loader.Location())
				data, err := ReadAll(loader)
				assert.Equal(testContents, string(data))
				assert.Nil(err)
			}

			if expander, err := factory.NewExpander(); assert.NotNil(expander) && assert.Nil(err) {
				if template, ok := expander.(*Template); assert.True(ok) {
					assert.Len(template.URITemplate.Names(), 0)
				}
			}

			expander, err := factory.NewExpander("key")
			assert.Nil(expander)
			assert.NotNil(err)
		}
	}
}

func TestFactoryHTTPLoader(t *testing.T) {
	assert := assert.New(t)
	factory := &Factory{
		URI: testFileURL,
	}

	if url, err := factory.URL(); assert.NotNil(url) && assert.Nil(err) {
		assert.Equal(HttpScheme, url.Scheme)
	}

	if loader, err := factory.NewLoader(); assert.NotNil(loader) && assert.Nil(err) {
		assert.Equal(testFileURL, loader.Location())
		data, err := ReadAll(loader)
		assert.Equal(testContents, string(data))
		assert.Nil(err)
	}

	if expander, err := factory.NewExpander(); assert.NotNil(expander) && assert.Nil(err) {
		if template, ok := expander.(*Template); assert.True(ok) {
			assert.Len(template.URITemplate.Names(), 0)
		}
	}

	expander, err := factory.NewExpander("key")
	assert.Nil(expander)
	assert.NotNil(err)
}
