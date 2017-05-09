package aws

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"
	"testing"
)

func testSNSMessage(scURL string) *SNSMessage {
	return &SNSMessage{
		Type:             "Notification",
		MessageId:        "thisismy-test-mess-agei-dentifier123",
		TopicArn:         "arn:aws:sns:us-east-1:000000000000:tester",
		Subject:          "Jello",
		Message:          "There is alway room for Jello",
		Timestamp:        time.Now().Format(time.RFC3339),
		SignatureVersion: "1",
		Signature:        "",
		SigningCertURL:   scURL,
		UnsubscribeURL:   "",
	}
}

func testCreateCerficate() (privkey *rsa.PrivateKey, pemkey []byte, err error) {
	template := &x509.Certificate {
		SignatureAlgorithm: x509.SHA1WithRSA,
		IsCA : true,
		BasicConstraintsValid : true,
		SubjectKeyId : []byte{11,22,33},
		SerialNumber : big.NewInt(1111),
		Subject : pkix.Name{
			Country : []string{"USA"},
			Organization: []string{"Comcast"},
		},
		NotBefore : time.Now(),
		NotAfter : time.Now().Add(time.Duration(5)*time.Second),
		ExtKeyUsage : []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage : x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign,
	}
	
	// generate private key
	privkey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	
	// create a self-signed certificate
	parent := template
	cert, err := x509.CreateCertificate(rand.Reader, template, parent, &privkey.PublicKey, privkey)
	
	pemkey = pem.EncodeToMemory(
		&pem.Block{
			Type: "CERTIFICATE",
			Bytes: cert,
		},
	)
	
	return
}

func testCreateSignature(privkey *rsa.PrivateKey, snsMsg *SNSMessage) (string, error) {
	formated := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n", 
		"Message", snsMsg.Message, 
		"MessageId", snsMsg.MessageId, 
		"Subject", snsMsg.Subject, 
		"Timestamp", snsMsg.Timestamp, 
		"TopicArn", snsMsg.TopicArn,
		"Type", snsMsg.Type,
	)
	
	h := sha1.Sum([]byte(formated))
	signature_b, err := rsa.SignPKCS1v15(rand.Reader, privkey, crypto.SHA1, h[:])
	
	return base64.StdEncoding.EncodeToString(signature_b), err
}

type testTransport struct {
	Transport http.RoundTripper
	URL       *url.URL
	Body      []byte
}

func(tt testTransport) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	r.URL = tt.URL

	resp = &http.Response{
		Header:     make(http.Header),
		Request:    r,
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       ioutil.NopCloser(bytes.NewBuffer(tt.Body)),
	}
	
	return
}

func testClient(u string, body []byte) (client *http.Client, err error) {	
	uu, err := url.Parse(u)
	if err != nil {
		return
	}
	
	tt := *new(testTransport)
	tt.URL = uu
	tt.Body = body
	
	client = &http.Client{Transport: tt}
	
	return
}

func testServer() *httptest.Server {
	h := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "test client response: %s\n", r.URL.String())
	}
	
	return httptest.NewServer(http.HandlerFunc(h))
}

func testCreateEnv() (pemkey []byte, server *httptest.Server, msgs map[string]*SNSMessage, err error) {
	privkey, pemkey, err := testCreateCerficate()
	if err != nil {
		return
	}
	
	server = testServer()
	
	snsMsg := testSNSMessage(server.URL)
	snsMsg.Signature, err = testCreateSignature(privkey, snsMsg)
	if err != nil {
		return
	}
	
	snsMsgGood := *snsMsg
	snsMsgBad  := *snsMsg
	snsMsgBad.Subject = "No more room for Jello"
	
	msgs = map[string]*SNSMessage{
		"good": &snsMsgGood,
		"bad": &snsMsgBad,
	}
	
	return
}

func Test_base64Decode(t *testing.T) {
	assert := assert.New(t)
	
	_, server, snsMsg, err := testCreateEnv()
	if server != nil {
		defer server.Close()
	}
	assert.Nil(err)
	
	_, errGood := base64Decode(snsMsg["good"])
	_, errBad  := base64Decode(snsMsg["bad"])
	
	assert.Nil(errGood)
	assert.Nil(errBad)
}

func Test_getPemFile(t *testing.T) {
	assert := assert.New(t)
	
	_, server, snsMsg, err := testCreateEnv()
	if server != nil {
		defer server.Close()
	}
	assert.Nil(err)
	
	client, err := testClient(server.URL, []byte(""))
	assert.Nil(err)
	
	v := NewValidator(client)
	
	_, errGood := v.getPemFile(snsMsg["good"].SigningCertURL)
	_, errBad  := v.getPemFile(snsMsg["bad"].SigningCertURL)
	
	assert.Nil(errGood)
	assert.Nil(errBad)
}

func Test_getCerticate(t *testing.T) {
	assert := assert.New(t)
	
	_, server, snsMsg, err := testCreateEnv()
	if server != nil {
		defer server.Close()
	}
	assert.Nil(err)
	
	client, err := testClient(server.URL, []byte(""))
	assert.Nil(err)
	
	v := NewValidator(client)
	
	pemFromGood, err := v.getPemFile(snsMsg["good"].SigningCertURL)
	assert.Nil(err)
	pemFromBad, err  := v.getPemFile(snsMsg["bad"].SigningCertURL)
	assert.Nil(err)

	_, errGood := getCerticate(pemFromGood)
	_, errBad  := getCerticate(pemFromBad)
	
	assert.Nil(errGood)
	assert.Nil(errBad)
}

func Test_generateSignature(t *testing.T) {
	assert := assert.New(t)
	
	snsMsg := testSNSMessage("127.0.0.1")
	h := generateSignature(snsMsg)
	
	assert.NotNil(h)
}

func Test_Validate(t *testing.T) {
	assert := assert.New(t)
	
	pemkey, server, snsMsg, err := testCreateEnv()
	if server != nil {
		defer server.Close()
	}
	assert.Nil(err)
	
	client, err := testClient(server.URL, pemkey)
	assert.Nil(err)
	
	v := NewValidator(client)
	
	okGood, errGood := v.Validate(snsMsg["good"])
	okBad, errBad := v.Validate(snsMsg["bad"])
	
	assert.True(okGood)
	assert.Nil(errGood)
	assert.False(okBad)
	assert.NotNil(errBad)
}
