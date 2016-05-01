package key

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	fileScheme  = "file"
	httpScheme  = "http"
	httpsScheme = "https"
)

// unsupportedKeyURL is an internal Factory Method for a formatted error
// indicating that a given URL is not a supported mechanism for retrieving
// key data
func unsupportedKeyURL(rawURL string) error {
	return errors.New(fmt.Sprintf("Unsupported key URL: %s", rawURL))
}

// IsSupportedKeyURL tests if the given url.URL instance is supported as a mechanism
// for retrieving keys.
func IsSupportedKeyURL(candidate *url.URL) bool {
	return candidate.Scheme == fileScheme || candidate.Scheme == httpScheme || candidate.Scheme == httpsScheme
}

// ParseKeyURL parses the given URL string into a url.URL.  If the URL is valid but
// is not supported as a source to obtain key data by this library, an error is returned.
func ParseKeyURL(rawURL string) (*url.URL, error) {
	url, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if !IsSupportedKeyURL(url) {
		return nil, unsupportedKeyURL(rawURL)
	}

	return url, nil
}

// Loader abstracts the notion of how key data is loaded.  A Loader provides
// two basic pieces of functionality:
// (1) Loading the raw key data from an arbitrary URL.  file:// URLs are supported,
// as are http:// and https://
// (2) Decoding and parsing the key, according to the Purpose.  Keys which are for
// verification and decryption must be public keys.  All others must be private keys.
type Loader interface {
	// Name returns the well-known name of the key returned from Load().  This name
	// is purely application-specific, and has nothing to do with the key itself.  It's the
	// identifier by which external code can refer to the loaded key.
	Name() string

	// Purpose returns the function of this key within the enclosing application.
	Purpose() Purpose

	// LoadKey obtains the raw key data and decodes it into one of the appropriate objects
	// returned from the crypto/x509 package.
	LoadKey() (interface{}, error)
}

// urlLoader is a simple, URL-based source of key data.  This is the core Loader implementation,
// and is always used.  It may, however, be decorated by a type below.
type urlLoader struct {
	name    string
	purpose Purpose
	url     url.URL
}

func (loader *urlLoader) String() string {
	return fmt.Sprintf("urlLoader{name: %s, purpose: %s, url: %s}", loader.name, loader.purpose, loader.url)
}

func (loader *urlLoader) Name() string {
	return loader.name
}

// Purpose returns the Purpose associated with this Loader.
func (loader *urlLoader) Purpose() Purpose {
	return loader.purpose
}

// URL returns the location from which this Loader gets its data
func (loader *urlLoader) URL() string {
	return loader.url.String()
}

// IsFile returns true if this Loader represents a file-based URL
func (loader *urlLoader) IsFile() bool {
	return loader.url.Scheme == fileScheme
}

// IsHttp returns true if this Loader represents an http-based URL,
// which includes both http and https.
func (loader *urlLoader) IsHttp() bool {
	return loader.url.Scheme == httpScheme || loader.url.Scheme == httpsScheme
}

// loadRawKeyData reads the raw, unmodified data from the internal url.
func (loader *urlLoader) loadRawKeyData() ([]byte, error) {
	if loader.IsFile() {
		return ioutil.ReadFile(loader.url.Path)
	} else if loader.IsHttp() {
		response, err := http.Get(loader.url.String())
		if response != nil {
			defer response.Body.Close()
		}

		if err != nil {
			return nil, err
		}

		return ioutil.ReadAll(response.Body)
	}

	return nil, unsupportedKeyURL(loader.url.String())
}

func (loader *urlLoader) LoadKey() (interface{}, error) {
	rawKeyData, err := loader.loadRawKeyData()
	if err != nil {
		return nil, err
	}

	pemBlock, _ := pem.Decode(rawKeyData)
	if pemBlock == nil {
		return nil, errors.New(fmt.Sprintf("Key from [%s] is not PEM encoded", loader.url))
	}

	switch loader.purpose {
	case PurposeVerify, PurposeDecrypt:
		return x509.ParsePKIXPublicKey(pemBlock.Bytes)

	default:
		return x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	}
}

// oneTimeLoader caches a key forever.  This type requires no
// other concurrency, and is mainly an optimization over cacheLoader
// for caches that never expire.
type oneTimeLoader struct {
	key      interface{}
	delegate Loader
}

func (loader *oneTimeLoader) String() string {
	return fmt.Sprintf("oneTimeLoader{delegate: %s}", loader.delegate)
}

func (loader *oneTimeLoader) Name() string {
	return loader.delegate.Name()
}

func (loader *oneTimeLoader) Purpose() Purpose {
	return loader.delegate.Purpose()
}

func (loader *oneTimeLoader) LoadKey() (interface{}, error) {
	return loader.key, nil
}

type cacheLoader struct {
	sync.RWMutex
	cachedKey   interface{}
	delegate    Loader
	cachePeriod time.Duration
	cacheExpiry time.Time
}

func (loader *cacheLoader) String() string {
	// this representation doesn't use anything guarded by the RW mutex
	return fmt.Sprintf("cacheLoader{delegate: %s, cachePeriod: %s}", loader.delegate, loader.cachePeriod)
}

func (loader *cacheLoader) Name() string {
	return loader.delegate.Name()
}

func (loader *cacheLoader) Purpose() Purpose {
	return loader.delegate.Purpose()
}

// unsafeIsCacheValid tests whether the currently cached key is still valid.
// This internal method must be called inside a mutex block.
func (loader *cacheLoader) unsafeIsCacheValid() bool {
	return time.Now().Before(loader.cacheExpiry)
}

// tryGetCachedKey contends on the loader's read lock and returns the
// cached key if the cached key hasn't expired.  This method returns nil
// if the cached key has expired.
func (loader *cacheLoader) tryGetCachedKey() interface{} {
	loader.RLock()
	defer loader.RUnlock()
	if loader.unsafeIsCacheValid() {
		return loader.cachedKey
	} else {
		return nil
	}
}

func (loader *cacheLoader) LoadKey() (interface{}, error) {
	key := loader.tryGetCachedKey()
	if key != nil {
		return key, nil
	}

	// upgrade to the write lock, which will require rechecking
	// the lock condition
	loader.Lock()
	defer loader.Unlock()
	if loader.unsafeIsCacheValid() {
		return loader.cachedKey, nil
	}

	newKey, err := loader.delegate.LoadKey()
	if err != nil {
		return nil, err
	}

	loader.cachedKey = newKey
	loader.cacheExpiry = time.Now().Add(loader.cachePeriod)
	return loader.cachedKey, nil
}

// LoaderBuilder implements both a builder for Loader instances and the
// external JSON representation of a Loader.
type LoaderBuilder struct {
	Name        string      `json:"name"`
	Url         string      `json:"url"`
	Purpose     Purpose     `json:"purpose"`
	CachePeriod CachePeriod `json:"cachePeriod"`
}

func (builder *LoaderBuilder) NewLoader() (Loader, error) {
	url, err := ParseKeyURL(builder.Url)
	if err != nil {
		return nil, err
	}

	urlLoader := &urlLoader{
		name:    builder.Name,
		purpose: builder.Purpose,
		url:     *url,
	}

	if builder.CachePeriod == CachePeriodNever {
		return urlLoader, nil
	}

	initialKey, err := urlLoader.LoadKey()
	if err != nil {
		return nil, err
	}

	if builder.CachePeriod > 0 {
		cachePeriod := time.Duration(builder.CachePeriod)

		return &cacheLoader{
			cachedKey:   initialKey,
			delegate:    urlLoader,
			cachePeriod: time.Duration(cachePeriod),
			cacheExpiry: time.Now().Add(cachePeriod),
		}, nil
	}

	// if set to default or forever ...
	return &oneTimeLoader{
		key:      initialKey,
		delegate: urlLoader,
	}, nil
}
