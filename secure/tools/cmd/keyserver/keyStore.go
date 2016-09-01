package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/Comcast/webpa-common/secure/key"
)

// KeyStore provides a single access point for a set of keys, keyed by their key identifiers
// or kid values in JWTs.
type KeyStore struct {
	keyIDs      []string
	privateKeys map[string]*rsa.PrivateKey
	publicKeys  map[string][]byte
}

func (ks *KeyStore) KeyIDs() []string {
	return ks.keyIDs
}

func (ks *KeyStore) PrivateKey(keyID string) (privateKey *rsa.PrivateKey, ok bool) {
	privateKey, ok = ks.privateKeys[keyID]
	return
}

func (ks *KeyStore) PublicKey(keyID string) (data []byte, ok bool) {
	data, ok = ks.publicKeys[keyID]
	return
}

// NewKeyStore exchanges a Configuration for a KeyStore.
func NewKeyStore(c *Configuration) (*KeyStore, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	privateKeys := make(map[string]*rsa.PrivateKey, len(c.Keys)+len(c.Generate))
	if err := resolveKeys(c, privateKeys); err != nil {
		return nil, err
	}

	if err := generateKeys(c, privateKeys); err != nil {
		return nil, err
	}

	publicKeys := make(map[string][]byte, len(privateKeys))
	if err := marshalPublicKeys(publicKeys, privateKeys); err != nil {
		return nil, err
	}

	keyIDs := make([]string, 0, len(privateKeys))
	for keyID, _ := range privateKeys {
		keyIDs = append(keyIDs, keyID)
	}

	return &KeyStore{
		keyIDs:      keyIDs,
		privateKeys: privateKeys,
		publicKeys:  publicKeys,
	}, nil
}

func resolveKeys(c *Configuration, privateKeys map[string]*rsa.PrivateKey) error {
	for keyID, resourceFactory := range c.Keys {
		keyResolver, err := (&key.ResolverFactory{
			Factory: *resourceFactory,
			Purpose: key.PurposeSign,
		}).NewResolver()

		if err != nil {
			return err
		}

		resolvedKey, err := keyResolver.ResolveKey(keyID)
		if err != nil {
			return err
		}

		if privateKey, ok := resolvedKey.(*rsa.PrivateKey); ok {
			privateKeys[keyID] = privateKey
		} else {
			return fmt.Errorf("The key %s did not resolve to an RSA private key")
		}
	}

	return nil
}

func generateKeys(c *Configuration, privateKeys map[string]*rsa.PrivateKey) error {
	bits := c.Bits
	if bits < 1 {
		bits = DefaultBits
	}

	for _, keyID := range c.Generate {
		if generatedKey, err := rsa.GenerateKey(rand.Reader, bits); err == nil {
			privateKeys[keyID] = generatedKey
		} else {
			return err
		}
	}

	return nil
}

func marshalPublicKeys(publicKeys map[string][]byte, privateKeys map[string]*rsa.PrivateKey) error {
	for keyID, privateKey := range privateKeys {
		derBytes, err := x509.MarshalPKIXPublicKey(privateKey.Public())
		if err != nil {
			return err
		}

		block := pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: derBytes,
		}

		var buffer bytes.Buffer
		err = pem.Encode(&buffer, &block)
		if err != nil {
			return err
		}

		publicKeys[keyID] = buffer.Bytes()
	}

	return nil
}
