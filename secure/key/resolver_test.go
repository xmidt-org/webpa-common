package key

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/webpa-common/v2/resource"
)

func TestSingleResolver(t *testing.T) {
	assert := assert.New(t)

	loader, err := (&resource.Factory{
		URI: publicKeyFilePath,
	}).NewLoader()

	if !assert.Nil(err) {
		return
	}

	expectedData, err := resource.ReadAll(loader)
	assert.NotEmpty(expectedData)
	assert.Nil(err)

	for _, purpose := range []Purpose{PurposeVerify, PurposeDecrypt, PurposeSign, PurposeEncrypt} {
		t.Logf("purpose: %s", purpose)

		expectedPair := &MockPair{}
		parser := &MockParser{}
		parser.On("ParseKey", purpose, expectedData).Return(expectedPair, nil).Once()

		var resolver Resolver = &singleResolver{
			basicResolver: basicResolver{
				parser:  parser,
				purpose: purpose,
			},

			loader: loader,
		}

		assert.Contains(fmt.Sprintf("%s", resolver), publicKeyFilePath)

		pair, err := resolver.ResolveKey("does not matter")
		assert.Equal(expectedPair, pair)
		assert.Nil(err)

		mock.AssertExpectationsForObjects(t, expectedPair.Mock, parser.Mock)
	}
}

func TestSingleResolverBadResource(t *testing.T) {
	assert := assert.New(t)

	var resolver Resolver = &singleResolver{
		basicResolver: basicResolver{
			parser:  DefaultParser,
			purpose: PurposeVerify,
		},
		loader: &resource.File{
			Path: "does not exist",
		},
	}

	key, err := resolver.ResolveKey("does not matter")
	assert.Nil(key)
	assert.NotNil(err)
}

func TestMultiResolver(t *testing.T) {
	assert := assert.New(t)

	expander, err := (&resource.Factory{
		URI: publicKeyFilePathTemplate,
	}).NewExpander()

	if !assert.Nil(err) {
		return
	}

	loader, err := expander.Expand(
		map[string]interface{}{KeyIdParameterName: keyId},
	)

	expectedData, err := resource.ReadAll(loader)
	assert.NotEmpty(expectedData)
	assert.Nil(err)

	for _, purpose := range []Purpose{PurposeVerify, PurposeDecrypt, PurposeSign, PurposeEncrypt} {
		t.Logf("purpose: %s", purpose)

		expectedPair := &MockPair{}
		parser := &MockParser{}
		parser.On("ParseKey", purpose, expectedData).Return(expectedPair, nil).Once()

		var resolver Resolver = &multiResolver{
			basicResolver: basicResolver{
				parser:  parser,
				purpose: purpose,
			},
			expander: expander,
		}

		assert.Contains(fmt.Sprintf("%s", resolver), publicKeyFilePathTemplate)

		pair, err := resolver.ResolveKey(keyId)
		assert.Equal(expectedPair, pair)
		assert.Nil(err)

		mock.AssertExpectationsForObjects(t, expectedPair.Mock, parser.Mock)
	}
}

func TestMultiResolverBadResource(t *testing.T) {
	assert := assert.New(t)

	var resolver Resolver = &multiResolver{
		expander: &resource.Template{
			URITemplate: resource.MustParse("/this/does/not/exist/{key}"),
		},
	}

	key, err := resolver.ResolveKey("this isn't valid")
	assert.Nil(key)
	assert.NotNil(err)
}

type badExpander struct {
	err error
}

func (bad *badExpander) Names() []string {
	return []string{}
}

func (bad *badExpander) Expand(interface{}) (resource.Loader, error) {
	return nil, bad.err
}

func TestMultiResolverBadExpander(t *testing.T) {
	assert := assert.New(t)

	expectedError := errors.New("The roof! The roof! The roof is on fire!")
	var resolver Resolver = &multiResolver{
		expander: &badExpander{expectedError},
	}

	key, err := resolver.ResolveKey("does not matter")
	assert.Nil(key)
	assert.Equal(expectedError, err)
}
