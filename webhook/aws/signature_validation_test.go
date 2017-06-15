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
	"testing"
	"time"
)

func testSNSMessage(scURL string) (*SNSMessage, *SNSMessage) {
	notification := &SNSMessage{
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

	token := "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123ab"
	arn := "arn:aws:sns:us-east-1:000000000000:tester"
	confirmation := &SNSMessage{
		Type:             "SubscriptionConfirmation",
		MessageId:        "thisismy-test-mess-agei-dentifier567",
		Token:            token,
		TopicArn:         arn,
		Message:          "You have chosen to subscribe to the topic arn:aws:sns:us-west-2:123456789012:MyTopic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
		SubscribeURL:     fmt.Sprintf("https://amazonawsaddress/?Action=ConfirmSubscription&TopicArn=%s&Token=%s", arn, token),
		Timestamp:        time.Now().Format(time.RFC3339),
		SignatureVersion: "1",
		Signature:        "",
		SigningCertURL:   scURL,
	}

	return notification, confirmation
}

func testCreateCerficate() (privkey *rsa.PrivateKey, pemkey []byte, err error) {
	template := &x509.Certificate{
		IsCA: true,
		BasicConstraintsValid: true,
		SubjectKeyId:          []byte{11, 22, 33},
		SerialNumber:          big.NewInt(1111),
		Subject: pkix.Name{
			Country:      []string{"USA"},
			Organization: []string{"Comcast"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Duration(5) * time.Second),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
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
			Type:  "CERTIFICATE",
			Bytes: cert,
		},
	)

	return
}

func testCreateSignature(privkey *rsa.PrivateKey, snsMsg *SNSMessage) (string, error) {
	fs, _ := formatSignature(snsMsg)
	h := sha1.Sum([]byte(fs))
	signature_b, err := rsa.SignPKCS1v15(rand.Reader, privkey, crypto.SHA1, h[:])

	return base64.StdEncoding.EncodeToString(signature_b), err
}

type testTransport struct {
	Transport http.RoundTripper
	URL       *url.URL
	Body      []byte
}

func (tt testTransport) RoundTrip(r *http.Request) (resp *http.Response, err error) {
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

	snsMsg_noti, snsMsg_conf := testSNSMessage(server.URL)
	snsMsg_noti.Signature, err = testCreateSignature(privkey, snsMsg_noti)
	if err != nil {
		return
	}
	snsMsg_conf.Signature, err = testCreateSignature(privkey, snsMsg_conf)
	if err != nil {
		return
	}

	snsMsgGood_noti := *snsMsg_noti
	snsMsgBad_noti := *snsMsg_noti
	snsMsgBad_noti.Subject = "No more room for Jello"

	snsMsgGood_conf := *snsMsg_conf
	snsMsgBad_conf := *snsMsg_conf
	snsMsgBad_conf.Message = "bad confirmation message"

	msgs = map[string]*SNSMessage{
		"noti-good": &snsMsgGood_noti,
		"noti-bad":  &snsMsgBad_noti,
		"conf-good": &snsMsgGood_conf,
		"conf-bad":  &snsMsgBad_conf,
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

	_, errGood := base64Decode(snsMsg["noti-good"])
	_, errBad := base64Decode(snsMsg["noti-bad"])

	assert.Nil(errGood)
	assert.Nil(errBad)

	_, errGood = base64Decode(snsMsg["conf-good"])
	_, errBad = base64Decode(snsMsg["conf-bad"])

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

	_, errGood := v.getPemFile(snsMsg["noti-good"].SigningCertURL)
	_, errBad := v.getPemFile(snsMsg["noti-bad"].SigningCertURL)

	assert.Nil(errGood)
	assert.Nil(errBad)

	_, errGood = v.getPemFile(snsMsg["conf-good"].SigningCertURL)
	_, errBad = v.getPemFile(snsMsg["conf-bad"].SigningCertURL)

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

	pemFromGood, err := v.getPemFile(snsMsg["noti-good"].SigningCertURL)
	assert.Nil(err)
	pemFromBad, err := v.getPemFile(snsMsg["noti-bad"].SigningCertURL)
	assert.Nil(err)

	_, errGood := getCerticate(pemFromGood)
	_, errBad := getCerticate(pemFromBad)

	assert.Nil(errGood)
	assert.Nil(errBad)

	pemFromGood, err = v.getPemFile(snsMsg["conf-good"].SigningCertURL)
	assert.Nil(err)
	pemFromBad, err = v.getPemFile(snsMsg["conf-bad"].SigningCertURL)
	assert.Nil(err)

	_, errGood = getCerticate(pemFromGood)
	_, errBad = getCerticate(pemFromBad)

	assert.Nil(errGood)
	assert.Nil(errBad)
}

func Test_formatSignature(t *testing.T) {
	assert := assert.New(t)

	snsMsg_noti, snsMsg_conf := testSNSMessage("127.0.0.1")
	fs1, _ := formatSignature(snsMsg_noti)
	fs2, _ := formatSignature(snsMsg_conf)

	assert.NotNil(fs1)
	assert.NotNil(fs2)
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

	okGood, errGood := v.Validate(snsMsg["noti-good"])
	okBad, errBad := v.Validate(snsMsg["noti-bad"])

	assert.True(okGood)
	assert.Nil(errGood)
	assert.False(okBad)
	assert.NotNil(errBad)

	okGood, errGood = v.Validate(snsMsg["conf-good"])
	okBad, errBad = v.Validate(snsMsg["conf-bad"])

	assert.True(okGood)
	assert.Nil(errGood)
	assert.False(okBad)
	assert.NotNil(errBad)
}
