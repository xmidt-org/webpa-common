package resource

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTemplateFile(t *testing.T) {
	assert := assert.New(t)

	for _, fileTemplate := range []string{testFilePathTemplate, testFileURITemplate} {
		t.Logf("fileTemplate: %s", fileTemplate)
		factory := &Factory{
			URI: fileTemplate,
		}

		templates := make([]*Template, 0, 2)
		if template, err := factory.NewTemplate(); assert.NotNil(template) && assert.Nil(err) {
			assert.Equal(fileTemplate, template.String())
			templates = append(templates, template)
		}

		if template, err := factory.NewTemplate(fileNameParameter); assert.NotNil(template) && assert.Nil(err) {
			assert.Equal(fileTemplate, template.String())
			templates = append(templates, template)
		}

		for _, template := range templates {
			values := map[string]interface{}{
				fileNameParameter: testFile,
			}

			if loader, err := template.Expand(values); assert.NotNil(loader) && assert.Nil(err) {
				data, err := ReadAll(loader)
				assert.Equal(testContents, string(data))
				assert.Nil(err)
			}

			loader, err := template.Expand(123)
			assert.Nil(loader)
			assert.NotNil(err)
		}

		template, err := factory.NewTemplate("nosuch")
		assert.Nil(template)
		assert.NotNil(err)
	}
}

func TestTemplateHTTP(t *testing.T) {
	assert := assert.New(t)
	factory := &Factory{
		URI: testFileURLTemplate,
	}

	templates := make([]*Template, 0, 2)
	if template, err := factory.NewTemplate(); assert.NotNil(template) && assert.Nil(err) {
		templates = append(templates, template)
	}

	if template, err := factory.NewTemplate(fileNameParameter); assert.NotNil(template) && assert.Nil(err) {
		templates = append(templates, template)
	}

	for _, template := range templates {
		values := map[string]interface{}{
			fileNameParameter: testFile,
		}

		if loader, err := template.Expand(values); assert.NotNil(loader) && assert.Nil(err) {
			data, err := ReadAll(loader)
			assert.Equal(testContents, string(data))
			assert.Nil(err)
		}

		loader, err := template.Expand(123)
		assert.Nil(loader)
		assert.NotNil(err)
	}

	template, err := factory.NewTemplate("nosuch")
	assert.Nil(template)
	assert.NotNil(err)
}
