package key

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSingleResolverPublicKey(t *testing.T) {
	assert := assert.New(t)

	for _, purpose := range []Purpose{PurposeVerify, PurposeDecrypt} {
		for _, keyURI := range []string{publicKeyFilePath, publicKeyURL} {
			t.Logf("purpose: %s, keyURI: %s", purpose, keyURI)

			loader, err := (&resource.Factory{
				URI: keyURI,
			}).NewLoader()

			if !assert.Nil(err) {
				continue
			}

			var resolver Resolver = &singleResolver{
				loader: loader,
				parser: purpose,
			}

			stringValue := fmt.Sprintf("%s", resolver)
			assert.Contains(stringValue, purpose.String())
			assert.Contains(stringValue, keyURI)

			key, err := resolver.ResolveKey("does not matter")
			assert.NotNil(key)
			assert.Nil(err)

			publicKey, ok := key.(*rsa.PublicKey)
			assert.NotNil(publicKey)
			assert.True(ok)
		}
	}
}

func TestSingleResolverPrivateKey(t *testing.T) {
	assert := assert.New(t)

	for _, purpose := range []Purpose{PurposeSign, PurposeEncrypt} {
		for _, keyURI := range []string{privateKeyFilePath, privateKeyURL} {
			t.Logf("purpose: %s, keyURI: %s", purpose, keyURI)

			loader, err := (&resource.Factory{
				URI: keyURI,
			}).NewLoader()

			if !assert.Nil(err) {
				continue
			}

			var resolver Resolver = &singleResolver{
				loader: loader,
				parser: purpose,
			}

			stringValue := fmt.Sprintf("%s", resolver)
			assert.Contains(stringValue, purpose.String())
			assert.Contains(stringValue, keyURI)

			key, err := resolver.ResolveKey("does not matter")
			assert.NotNil(key)
			assert.Nil(err)

			privateKey, ok := key.(*rsa.PrivateKey)
			assert.NotNil(privateKey)
			assert.True(ok)
		}
	}
}

func TestSingleResolverBadResource(t *testing.T) {
	assert := assert.New(t)

	var resolver Resolver = &singleResolver{
		loader: &resource.File{
			Path: "does not exist",
		},
		parser: PurposeVerify,
	}

	key, err := resolver.ResolveKey("does not matter")
	assert.Nil(key)
	assert.NotNil(err)
}

func TestMultiResolverPublicKey(t *testing.T) {
	assert := assert.New(t)

	for _, purpose := range []Purpose{PurposeVerify, PurposeDecrypt} {
		for _, keyURITemplate := range []string{publicKeyFilePathTemplate, publicKeyURLTemplate} {
			t.Logf("purpose: %s, keyURITemplate: %s", purpose, keyURITemplate)

			expander, err := (&resource.Factory{
				URI: keyURITemplate,
			}).NewExpander()

			if !assert.Nil(err) {
				continue
			}

			var resolver Resolver = &multiResolver{
				expander: expander,
				parser:   purpose,
			}

			stringValue := fmt.Sprintf("%s", resolver)
			assert.Contains(stringValue, purpose.String())
			assert.Contains(stringValue, keyURITemplate)

			key, err := resolver.ResolveKey(keyId)
			assert.NotNil(key)
			assert.Nil(err)

			publicKey, ok := key.(*rsa.PublicKey)
			assert.NotNil(publicKey)
			assert.True(ok)
		}
	}
}

func TestMultiResolverPrivateKey(t *testing.T) {
	assert := assert.New(t)

	for _, purpose := range []Purpose{PurposeSign, PurposeEncrypt} {
		for _, keyURITemplate := range []string{privateKeyFilePathTemplate, privateKeyURLTemplate} {
			t.Logf("purpose: %s, keyURITemplate: %s", purpose, keyURITemplate)

			expander, err := (&resource.Factory{
				URI: keyURITemplate,
			}).NewExpander()

			if !assert.Nil(err) {
				continue
			}

			var resolver Resolver = &multiResolver{
				expander: expander,
				parser:   purpose,
			}

			stringValue := fmt.Sprintf("%s", resolver)
			assert.Contains(stringValue, purpose.String())
			assert.Contains(stringValue, keyURITemplate)

			key, err := resolver.ResolveKey(keyId)
			assert.NotNil(key)
			assert.Nil(err)

			privateKey, ok := key.(*rsa.PrivateKey)
			assert.NotNil(privateKey)
			assert.True(ok)
		}
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
