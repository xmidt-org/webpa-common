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
				basicResolver: basicResolver{
					parser:  DefaultParser,
					purpose: purpose,
				},

				loader: loader,
			}

			stringValue := fmt.Sprintf("%s", resolver)
			assert.Contains(stringValue, purpose.String())
			assert.Contains(stringValue, keyURI)

			pair, err := resolver.ResolveKey("does not matter")
			assert.NotNil(pair)
			assert.Nil(err)

			publicKey, ok := pair.Public().(*rsa.PublicKey)
			assert.NotNil(publicKey)
			assert.True(ok)

			assert.False(pair.HasPrivate())
			assert.Nil(pair.Private())
			assert.Equal(purpose, pair.Purpose())
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
				basicResolver: basicResolver{
					parser:  DefaultParser,
					purpose: purpose,
				},
				loader: loader,
			}

			stringValue := fmt.Sprintf("%s", resolver)
			assert.Contains(stringValue, purpose.String())
			assert.Contains(stringValue, keyURI)

			pair, err := resolver.ResolveKey("does not matter")
			assert.NotNil(pair)
			assert.Nil(err)

			publicKey, ok := pair.Public().(*rsa.PublicKey)
			assert.NotNil(publicKey)
			assert.True(ok)

			assert.True(pair.HasPrivate())
			assert.Equal(purpose, pair.Purpose())

			privateKey, ok := pair.Private().(*rsa.PrivateKey)
			assert.NotNil(privateKey)
		}
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
				basicResolver: basicResolver{
					parser:  DefaultParser,
					purpose: purpose,
				},
				expander: expander,
			}

			stringValue := fmt.Sprintf("%s", resolver)
			assert.Contains(stringValue, purpose.String())
			assert.Contains(stringValue, keyURITemplate)

			pair, err := resolver.ResolveKey(keyId)
			assert.NotNil(pair)
			assert.Nil(err)

			publicKey, ok := pair.Public().(*rsa.PublicKey)
			assert.NotNil(publicKey)
			assert.True(ok)

			assert.False(pair.HasPrivate())
			assert.Nil(pair.Private())
			assert.Equal(purpose, pair.Purpose())
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
				basicResolver: basicResolver{
					parser:  DefaultParser,
					purpose: purpose,
				},
				expander: expander,
			}

			stringValue := fmt.Sprintf("%s", resolver)
			assert.Contains(stringValue, purpose.String())
			assert.Contains(stringValue, keyURITemplate)

			pair, err := resolver.ResolveKey(keyId)
			assert.NotNil(pair)
			assert.Nil(err)

			publicKey, ok := pair.Public().(*rsa.PublicKey)
			assert.NotNil(publicKey)
			assert.True(ok)

			assert.Equal(purpose, pair.Purpose())
			assert.True(pair.HasPrivate())

			privateKey, ok := pair.Private().(*rsa.PrivateKey)
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
