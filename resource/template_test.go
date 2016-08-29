package resource

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMustParseValidTemplate(t *testing.T) {
	assert := assert.New(t)

	if template := MustParse("/etc/{key}.txt"); assert.NotNil(template) {
		result, err := template.Expand(map[string]interface{}{"key": "value"})
		assert.Equal(result, "/etc/value.txt")
		assert.Nil(err)
	}
}

func TestMustParseInvalidTemplate(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.NotNil(r)
	}()

	MustParse("/invalid/{bad")
}

func TestTemplateFile(t *testing.T) {
	assert := assert.New(t)

	for _, fileTemplate := range []string{testFilePathTemplate, testFileURITemplate} {
		t.Logf("fileTemplate: %s", fileTemplate)
		factory := &Factory{
			URI: fileTemplate,
		}

		expanders := make([]Expander, 0, 2)
		if expander, err := factory.NewExpander(); assert.NotNil(expander) && assert.Nil(err) {
			assert.Equal(fileTemplate, fmt.Sprintf("%s", expander))
			expanders = append(expanders, expander)
		}

		if expander, err := factory.NewExpander(fileNameParameter); assert.NotNil(expander) && assert.Nil(err) {
			assert.Equal(fileTemplate, fmt.Sprintf("%s", expander))
			expanders = append(expanders, expander)
		}

		for _, expander := range expanders {
			names := expander.Names()
			assert.Len(names, 1)
			assert.Equal(fileNameParameter, names[0])

			values := map[string]interface{}{
				fileNameParameter: testFile,
			}

			if loader, err := expander.Expand(values); assert.NotNil(loader) && assert.Nil(err) {
				data, err := ReadAll(loader)
				assert.Equal(testContents, string(data))
				assert.Nil(err)
			}

			loader, err := expander.Expand(123)
			assert.Nil(loader)
			assert.NotNil(err)
		}

		expander, err := factory.NewExpander("nosuch")
		assert.Nil(expander)
		assert.NotNil(err)
	}
}

func TestTemplateHTTP(t *testing.T) {
	assert := assert.New(t)
	factory := &Factory{
		URI: testFileURLTemplate,
	}

	expanders := make([]Expander, 0, 2)
	if expander, err := factory.NewExpander(); assert.NotNil(expander) && assert.Nil(err) {
		expanders = append(expanders, expander)
	}

	if expander, err := factory.NewExpander(fileNameParameter); assert.NotNil(expander) && assert.Nil(err) {
		expanders = append(expanders, expander)
	}

	for _, expander := range expanders {
		names := expander.Names()
		assert.Len(names, 1)
		assert.Equal(fileNameParameter, names[0])

		values := map[string]interface{}{
			fileNameParameter: testFile,
		}

		if loader, err := expander.Expand(values); assert.NotNil(loader) && assert.Nil(err) {
			data, err := ReadAll(loader)
			assert.Equal(testContents, string(data))
			assert.Nil(err)
		}

		loader, err := expander.Expand(123)
		assert.Nil(loader)
		assert.NotNil(err)
	}

	expander, err := factory.NewExpander("nosuch")
	assert.Nil(expander)
	assert.NotNil(err)
}
