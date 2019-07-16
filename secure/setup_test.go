package secure

import (
	"fmt"
	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
	"github.com/xmidt-org/webpa-common/resource"
	"github.com/xmidt-org/webpa-common/secure/key"
	"os"
	"testing"
)

const (
	publicKeyFileName  = "jwt-key.pub"
	privateKeyFileName = "jwt-key"
)

var (
	publicKeyFileURI  string
	publicKeyResolver key.Resolver

	privateKeyFileURI  string
	privateKeyResolver key.Resolver

	// ripped these test claims from the SATS swagger example
	testClaims = jws.Claims{
		"valid":        true,
		"capabilities": []interface{}{"x1:webpa:api:.*:post"},
		"allowedResources": map[string]interface{}{
			"allowedDeviceIds":         []interface{}{"1641529834193109183"},
			"allowedPartners":          []interface{}{"comcast, cox"},
			"allowedServiceAccountIds": []interface{}{"4924346887352567847"},
		},
	}

	testJWT           jwt.JWT
	testSerializedJWT []byte
)

func TestMain(m *testing.M) {
	os.Exit(func() int {
		currentDirectory, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to obtain current working directory: %s\n", err)
			return 1
		}

		publicKeyFileURI = fmt.Sprintf("%s/%s", currentDirectory, publicKeyFileName)
		privateKeyFileURI = fmt.Sprintf("%s/%s", currentDirectory, privateKeyFileName)

		privateKeyResolver, err = (&key.ResolverFactory{
			Factory: resource.Factory{URI: privateKeyFileURI},
			Purpose: key.PurposeSign,
		}).NewResolver()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create private key resolver: %s\n", err)
			return 1
		}

		publicKeyResolver, err = (&key.ResolverFactory{
			Factory: resource.Factory{URI: publicKeyFileURI},
			Purpose: key.PurposeVerify,
		}).NewResolver()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create public key resolver: %s\n", err)
			return 1
		}

		pair, err := privateKeyResolver.ResolveKey("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to resolve private key: %s\n", err)
			return 1
		}

		// generate a unique JWT for each run of the tests
		// this also exercises our secure/key infrastructure
		testJWT = jws.NewJWT(testClaims, crypto.SigningMethodRS256)
		testSerializedJWT, err = testJWT.Serialize(pair.Private())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to serialize test JWT: %s\n", err)
			return 1
		}

		return m.Run()
	}())
}
