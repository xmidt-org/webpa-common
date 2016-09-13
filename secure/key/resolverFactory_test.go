package key

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/resource"
	"github.com/Comcast/webpa-common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func ExampleSingleKeyConfiguration() {
	jsonConfiguration := fmt.Sprintf(`{
		"uri": "%s",
		"purpose": "verify",
		"header": {
			"Accept": ["text/plain"]
		}
	}`, publicKeyURL)

	var factory ResolverFactory
	if err := json.Unmarshal([]byte(jsonConfiguration), &factory); err != nil {
		fmt.Println(err)
		return
	}

	resolver, err := factory.NewResolver()
	if err != nil {
		fmt.Println(err)
		return
	}

	// althrough we pass a keyId, it doesn't matter
	// the keyId would normally come from a JWT or other source, but
	// this configuration maps all key identifiers onto the same resource
	key, err := resolver.ResolveKey(keyId)
	if err != nil {
		fmt.Println(err)
		return
	}

	publicKey, ok := key.Public().(*rsa.PublicKey)
	if !ok {
		fmt.Println("Expected a public key")
	}

	fmt.Printf("%#v", publicKey)

	// Output:
	// &rsa.PublicKey{N:27943075365309976493653163303797959212418241538912650140443307384472696765226993413692820781465849081859025776428168351053450151991381458393395627926945090025279037554792902370352660829719944448435879538779506598037701785142079839040587119599241554109043386957121126327267661933261531301157240649436180239359321477795441956911062536999488590278721548425004681839069551715529565117581358421070795577996947939534909344145027536788621293233751031126681790089555592380957432236272148722403554429033227913702251021698422165616430378445527162280875770582636410571931829939754369601100687471071175959731316949515587341982201, E:65537}
}

func ExampleURITemplateConfiguration() {
	jsonConfiguration := fmt.Sprintf(`{
		"uri": "%s",
		"purpose": "verify",
		"header": {
			"Accept": ["text/plain"]
		}
	}`, publicKeyURLTemplate)

	var factory ResolverFactory
	if err := json.Unmarshal([]byte(jsonConfiguration), &factory); err != nil {
		fmt.Println(err)
		return
	}

	resolver, err := factory.NewResolver()
	if err != nil {
		fmt.Println(err)
		return
	}

	key, err := resolver.ResolveKey(keyId)
	if err != nil {
		fmt.Println(err)
		return
	}

	publicKey, ok := key.Public().(*rsa.PublicKey)
	if !ok {
		fmt.Println("Expected a public key")
	}

	fmt.Printf("%#v", publicKey)

	// Output:
	// &rsa.PublicKey{N:27943075365309976493653163303797959212418241538912650140443307384472696765226993413692820781465849081859025776428168351053450151991381458393395627926945090025279037554792902370352660829719944448435879538779506598037701785142079839040587119599241554109043386957121126327267661933261531301157240649436180239359321477795441956911062536999488590278721548425004681839069551715529565117581358421070795577996947939534909344145027536788621293233751031126681790089555592380957432236272148722403554429033227913702251021698422165616430378445527162280875770582636410571931829939754369601100687471071175959731316949515587341982201, E:65537}
}

func TestBadURITemplates(t *testing.T) {
	assert := assert.New(t)

	badURITemplates := []string{
		"",
		"badscheme://foo/bar.pem",
		"http://badtemplate.com/{bad",
		"file:///etc/{too}/{many}/{parameters}",
		"http://missing.keyId.com/{someOtherName}",
	}

	for _, badURITemplate := range badURITemplates {
		t.Logf("badURITemplate: %s", badURITemplate)

		factory := ResolverFactory{
			Factory: resource.Factory{
				URI: badURITemplate,
			},
		}

		resolver, err := factory.NewResolver()
		assert.Nil(resolver)
		assert.NotNil(err)
	}
}

func TestResolverFactoryNewUpdater(t *testing.T) {
	assert := assert.New(t)

	updateKeysCalled := make(chan struct{})
	runner := func(mock.Arguments) {
		defer func() {
			recover() // ignore panics from multiple closes
		}()

		close(updateKeysCalled)
	}

	keyCache := &MockCache{}
	keyCache.On("UpdateKeys").Return(0, nil).Run(runner)

	resolverFactory := ResolverFactory{
		UpdateInterval: types.Duration(100 * time.Millisecond),
	}

	if updater := resolverFactory.NewUpdater(keyCache); assert.NotNil(updater) {
		waitGroup := &sync.WaitGroup{}
		shutdown := make(chan struct{})
		updater.Run(waitGroup, shutdown)

		// we only care that the updater called UpdateKeys() at least once
		<-updateKeysCalled
		close(shutdown)
		waitGroup.Wait()
	}
}

func TestResolverFactoryDefaultParser(t *testing.T) {
	assert := assert.New(t)

	parser := &MockParser{}
	resolverFactory := ResolverFactory{
		Factory: resource.Factory{
			URI: publicKeyFilePath,
		},
	}

	assert.Equal(DefaultParser, resolverFactory.parser())
	mock.AssertExpectationsForObjects(t, parser.Mock)
}

func TestResolverFactoryCustomParser(t *testing.T) {
	assert := assert.New(t)

	parser := &MockParser{}
	resolverFactory := ResolverFactory{
		Factory: resource.Factory{
			URI: publicKeyFilePath,
		},
		Parser: parser,
	}

	assert.Equal(parser, resolverFactory.parser())
	mock.AssertExpectationsForObjects(t, parser.Mock)
}
