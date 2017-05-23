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
/*	
	token := "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123ab"
	arn   := "arn:aws:sns:us-east-1:000000000000:tester"
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
*/
/*	// from shilpa
	confirmation := &SNSMessage{
		Type: "SubscriptionConfirmation",
		MessageId: "631d8695-73c2-475c-8fe6-4b28315be625",
		Token: "2336412f37fb687f5d51e6e241d59b68c8bc222ad6fa9ea81821ca3d0d8b34d297afec055201b481ae221790a4b4ad641a35407f07edd91c7e178e73e38cce35b7904e19a2b91e1bfed1022c64d611f46a6d463bf7dd2ba340a37bbc5a7a2b6f1c5b47b6445504484d166ce8f609045d",
		TopicArn: "arn:aws:sns:us-east-1:801199994599:test-topic",
		Message: "You have chosen to subscribe to the topic arn:aws:sns:us-east-1:801199994599:test-topic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
		SubscribeURL: "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-east-1:801199994599:test-topic&Token=2336412f37fb687f5d51e6e241d59b68c8bc222ad6fa9ea81821ca3d0d8b34d297afec055201b481ae221790a4b4ad641a35407f07edd91c7e178e73e38cce35b7904e19a2b91e1bfed1022c64d611f46a6d463bf7dd2ba340a37bbc5a7a2b6f1c5b47b6445504484d166ce8f609045d",
		Timestamp: "2017-05-17T02:21:40.411Z",
		SignatureVersion: "1",
		Signature: "H1tuVP298v6QM2Eupm5fz+iYuMoAleIHCDVuZLd+A2h9gwy7r+LuL5/OMIXtUAY/A8tooofBi6Bs9y60WhQv8ktsLlptrJugRslyObIE4npal5PvROur+HS3bBSBZdcyfWZwAIxrlimkMkd1t+A+SBHU//41t1ulJYCiqOOokP/SpiTmu1GHMpXm13oCAG990I4LDXkL4vidulk7njAUzoCjkNsFWc3gERSsofFmxDe5l5SuN7bYSPwFqMOpedVoLbp6g+AwddJkXsPINcWZx7FT7taoeHI6CXDeK1Xx4Ay1LqNSpKLK8giO2cQ7Al7etIMYeumfJqt8Z7pG+t1Edw==",
		SigningCertURL: "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-b95095beb82e8f6a046b3aafc7f4149a.pem",
	}
*/	
/*
	// from debug message
	var ma map[string]MsgAttr
	confirmation := &SNSMessage{
		Type: "SubscriptionConfirmation",
		MessageId: "1f526c86-20e4-4912-ae7f-125b5c8b3c25",
		Token: "2336412f37fb687f5d51e6e241d59b68c8bc22259cff35774f038e302ed774e3401efee3e79fbbd8278273dc4a0e2415221b8503a4c525dec14635c1f0a2f536fc7fde000994c7327bb4a94fdc97398f2545a83c63a5b1632b9c160eecde0df5186f98e59e7c3b1672844b71a16245a7",
		TopicArn: "arn:aws:sns:us-east-1:801199994599:test-topic",
		Message: "You have chosen to subscribe to the topic arn:aws:sns:us-east-1:801199994599:test-topic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
		SubscribeURL: "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-east-1:801199994599:test-topic&Token=2336412f37fb687f5d51e6e241d59b68c8bc22259cff35774f038e302ed774e3401efee3e79fbbd8278273dc4a0e2415221b8503a4c525dec14635c1f0a2f536fc7fde000994c7327bb4a94fdc97398f2545a83c63a5b1632b9c160eecde0df5186f98e59e7c3b1672844b71a16245a7",
		Timestamp: "2017-05-18T18:40:29.067Z",
		SignatureVersion: "1",
		Signature: "OIM0q4r/F+yvW75h8+LXQ/51jFw7ENV0E5LXG0OHHPchsCVz98V5lz1yb6qUZFjXxX1GAEkZ0IMbLY/CvtxhlOY+LRIxb7DH9E3Q0fxU0746I7f/8zNeplOtipnpSK57a+DLm6cHCJA5IXTFOJeg7nhs2zmdsSKLk1/WEVGFjSZGaKLapoiXdKJIf9UI77O1x5/mWjkz7hGa0pOym2j/UbX0Gas2JOA8dNc6lLkpDl7ykouNiDAdKyWxU+BLs7slZuYL7UGUPKajYeCMU/5SIHiSJh1Bu+Ti+FplXNQrdiDkhloaGOJ1sTTQt4eAT8Nh25AkBsF/3KRmy5KQds3Obw==",
		SigningCertURL: "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-b95095beb82e8f6a046b3aafc7f4149a.pem",
		UnsubscribeURL: "",
		MessageAttributes: ma,
	}
*/
/*
pub: &{N:+19382921740755029201441068501853084389131754448056561445939179934772179366728845534842263725116349554888243183651450625154513309480733486755650158829395739048148695751191890309902866773171765916198437549012580072019587905449674869757597703154070959893462984096223721720558808658843059287204185561466711948197748067512410829674849962524788529768774314168509706690024323022452390042303262872788691004632848449475852321535586571469964679521605002870530433942179021582816446233302637508750071947835950077349299269933071361957236779738308959979530319998918649832141178647560137557010127016581815914614602744882319388741997 E:65537}
, h.Sum(nil): [213 255 17 63 154 76 254 237 169 116 109 230 192 10 255 147 19 79 212 6]
decodedSignature: [69 9 186 241 222 55 156 237 78 197 160 230 123 201 57 95 36 111 88 184 49 193 238 72 104 71 76 142 43 170 219 33 213 164 90 36 102 79 217 134 105 126 126 255 202 240 38 177 235 189 254 90 110 197 185 244 50 253 191 45 140 81 57 223 60 93 61 95 150 166 41 215 180 32 195 110 79 70 184 88 94 236 177 229 55 184 3 11 146 224 162 176 144 125 102 47 79 217 117 73 3 38 49 90 60 24 168 252 164 76 112 185 112 225 87 30 134 13 19 203 136 30 163 118 220 43 126 222 152 174 163 160 43 147 153 164 213 148 185 60 84 87 184 178 28 57 225 63 24 221 8 108 29 133 240 108 222 0 77 180 160 187 141 244 31 101 206 193 52 248 73 199 166 155 12 31 235 117 192 211 161 7 101 1 17 211 185 142 84 8 223 196 210 104 124 63 55 135 197 127 103 131 161 58 0 30 157 107 222 164 205 130 138 185 18 238 73 23 91 96 14 181 80 32 255 169 107 24 196 91 49 214 94 223 189 226 86 249 180 147 255 152 101 128 8 161 186 6 60 181 188 78 240 73 198 5]
*/
/*	// another debug message
	var ma map[string]MsgAttr
	confirmation := &SNSMessage{
		Type: "SubscriptionConfirmation",
		MessageId: "a034b92c-229b-43a6-826e-a93797a8aba6",
		Token: "2336412f37fb687f5d51e6e241d59b68c8bc22259cfc6e5f04cbeaed6cf5d6a7e511b62241b3b272370517637780c475e8d0f3852dcfba1e49baecaf32c7cb7551e382d1a5a8cfa3b72249f91346128786ed6963e764c2593d2b8d0b554149345606866190330359fed37abc7eda3aaa",
		TopicArn: "arn:aws:sns:us-east-1:801199994599:test-topic",
		Subject: "",
		Message: "You have chosen to subscribe to the topic arn:aws:sns:us-east-1:801199994599:test-topic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
		SubscribeURL: "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-east-1:801199994599:test-topic&Token=2336412f37fb687f5d51e6e241d59b68c8bc22259cfc6e5f04cbeaed6cf5d6a7e511b62241b3b272370517637780c475e8d0f3852dcfba1e49baecaf32c7cb7551e382d1a5a8cfa3b72249f91346128786ed6963e764c2593d2b8d0b554149345606866190330359fed37abc7eda3aaa",
		Timestamp: "2017-05-18T20:06:25.371Z",
		SignatureVersion: "1",
		Signature: "RQm68d43nO1OxaDme8k5XyRvWLgxwe5IaEdMjiuq2yHVpFokZk/Zhml+fv/K8Cax673+Wm7FufQy/b8tjFE53zxdPV+WpinXtCDDbk9GuFhe7LHlN7gDC5LgorCQfWYvT9l1SQMmMVo8GKj8pExwuXDhVx6GDRPLiB6jdtwrft6YrqOgK5OZpNWUuTxUV7iyHDnhPxjdCGwdhfBs3gBNtKC7jfQfZc7BNPhJx6abDB/rdcDToQdlARHTuY5UCN/E0mh8PzeHxX9ng6E6AB6da96kzYKKuRLuSRdbYA61UCD/qWsYxFsx1l7fveJW+bST/5hlgAihugY8tbxO8EnGBQ==",
		SigningCertURL: "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-b95095beb82e8f6a046b3aafc7f4149a.pem",
		UnsubscribeURL: "",
		MessageAttributes: ma,
	}
*/

/*
pub: &{N:+19382921740755029201441068501853084389131754448056561445939179934772179366728845534842263725116349554888243183651450625154513309480733486755650158829395739048148695751191890309902866773171765916198437549012580072019587905449674869757597703154070959893462984096223721720558808658843059287204185561466711948197748067512410829674849962524788529768774314168509706690024323022452390042303262872788691004632848449475852321535586571469964679521605002870530433942179021582816446233302637508750071947835950077349299269933071361957236779738308959979530319998918649832141178647560137557010127016581815914614602744882319388741997 E:65537}
, h: [77 101 115 115 97 103 101 10 89 111 117 32 104 97 118 101 32 99 104 111 115 101 110 32 116 111 32 115 117 98 115 99 114 105 98 101 32 116 111 32 116 104 101 32 116 111 112 105 99 32 97 114 110 58 97 119 115 58 115 110 115 58 117 115 45 101 97 115 116 45 49 58 56 48 49 49 57 57 57 57 52 53 57 57 58 116 101 115 116 45 116 111 112 105 99 46 10 84 111 32 99 111 110 102 105 114 109 32 116 104 101 32 115 117 98 115 99 114 105 112 116 105 111 110 44 32 118 105 115 105 116 32 116 104 101 32 83 117 98 115 99 114 105 98 101 85 82 76 32 105 110 99 108 117 100 101 100 32 105 110 32 116 104 105 115 32 109 101 115 115 97 103 101 46 10 77 101 115 115 97 103 101 73 100 10 55 54 102 97 55 99 51 55 45 57 54 57 99 45 52 53 56 97 45 57 100 55 53 45 49 48 101 98 55 55 102 56 100 48 99 54 10 83 117 98 115 99 114 105 98 101 85 82 76 10 104 116 116 112 115 58 47 47 115 110 115 46 117 115 45 101 97 115 116 45 49 46 97 109 97 122 111 110 97 119 115 46 99 111 109 47 63 65 99 116 105 111 110 61 67 111 110 102 105 114 109 83 117 98 115 99 114 105 112 116 105 111 110 38 84 111 112 105 99 65 114 110 61 97 114 110 58 97 119 115 58 115 110 115 58 117 115 45 101 97 115 116 45 49 58 56 48 49 49 57 57 57 57 52 53 57 57 58 116 101 115 116 45 116 111 112 105 99 38 84 111 107 101 110 61 50 51 51 54 52 49 50 102 51 55 102 98 54 56 55 102 53 100 53 49 101 54 101 50 52 49 100 53 57 98 54 56 99 56 98 99 50 50 50 53 57 99 102 99 54 102 54 57 51 102 100 56 51 57 54 55 100 53 52 102 56 102 101 53 49 101 51 101 100 55 57 54 54 54 101 48 99 56 56 53 97 53 57 98 49 98 56 98 57 53 52 55 53 99 57 49 52 100 52 100 100 52 101 101 57 99 102 49 101 49 101 51 99 52 50 99 99 101 57 51 55 100 51 52 48 52 57 50 100 97 54 98 54 49 102 54 101 54 102 97 54 101 101 100 97 101 51 99 53 48 54 99 52 98 51 52 53 52 98 55 49 102 53 99 98 97 98 53 102 50 49 53 101 53 54 56 53 56 56 99 49 56 52 100 49 53 101 98 48 102 98 98 50 53 99 57 100 57 97 56 50 100 48 53 53 100 98 52 52 101 56 101 97 52 100 99 100 56 99 100 57 51 50 101 10 84 105 109 101 115 116 97 109 112 10 37 33 40 69 88 84 82 65 32 115 116 114 105 110 103 61 50 48 49 55 45 48 53 45 49 56 84 50 49 58 48 48 58 51 57 46 52 52 57 90 44 32 115 116 114 105 110 103 61 84 111 107 101 110 44 32 115 116 114 105 110 103 61 50 51 51 54 52 49 50 102 51 55 102 98 54 56 55 102 53 100 53 49 101 54 101 50 52 49 100 53 57 98 54 56 99 56 98 99 50 50 50 53 57 99 102 99 54 102 54 57 51 102 100 56 51 57 54 55 100 53 52 102 56 102 101 53 49 101 51 101 100 55 57 54 54 54 101 48 99 56 56 53 97 53 57 98 49 98 56 98 57 53 52 55 53 99 57 49 52 100 52 100 100 52 101 101 57 99 102 49 101 49 101 51 99 52 50 99 99 101 57 51 55 100 51 52 48 52 57 50 100 97 54 98 54 49 102 54 101 54 102 97 54 101 101 100 97 101 51 99 53 48 54 99 52 98 51 52 53 52 98 55 49 102 53 99 98 97 98 53 102 50 49 53 101 53 54 56 53 56 56 99 49 56 52 100 49 53 101 98 48 102 98 98 50 53 99 57 100 57 97 56 50 100 48 53 53 100 98 52 52 101 56 101 97 52 100 99 100 56 99 100 57 51 50 101 44 32 115 116 114 105 110 103 61 84 111 112 105 99 65 114 110 44 32 115 116 114 105 110 103 61 97 114 110 58 97 119 115 58 115 110 115 58 117 115 45 101 97 115 116 45 49 58 56 48 49 49 57 57 57 57 52 53 57 57 58 116 101 115 116 45 116 111 112 105 99 44 32 115 116 114 105 110 103 61 84 121 112 101 44 32 115 116 114 105 110 103 61 83 117 98 115 99 114 105 112 116 105 111 110 67 111 110 102 105 114 109 97 116 105 111 110 41]
decodedSignature: [77 16 197 144 232 151 55 208 105 43 45 105 237 191 163 73 150 183 119 248 47 14 140 221 116 234 103 67 220 145 146 100 197 95 131 168 52 34 232 227 17 204 94 57 178 108 30 132 148 165 247 179 8 183 103 157 94 131 158 52 0 35 139 166 188 178 24 187 110 110 213 121 39 96 90 45 191 223 196 59 170 38 51 67 51 45 92 150 101 33 121 146 149 67 89 203 13 57 156 117 201 18 229 186 146 3 213 123 74 219 113 167 25 94 171 28 36 151 190 254 117 224 150 161 159 75 248 46 230 83 95 82 105 143 1 236 134 174 227 191 192 75 223 96 133 181 60 85 78 57 254 7 13 178 176 238 2 45 159 133 61 157 23 132 69 63 39 204 220 77 12 3 163 17 194 172 35 246 140 95 251 119 72 1 144 200 63 123 41 117 196 237 20 160 61 124 112 14 34 105 211 60 193 40 227 184 115 74 41 68 49 108 248 50 103 192 60 112 52 30 120 122 79 162 8 21 220 187 229 24 77 134 176 63 152 93 202 153 248 249 95 13 7 36 174 167 216 177 252 113 22 111 236 115 9 18]
signature validation failed [false]: crypto/rsa: verification error
[ERROR] SNS signature validation error crypto/rsa: verification error
[DEBUG] SNS signature validation error with subscription message &{Type:SubscriptionConfirmation MessageId:76fa7c37-969c-458a-9d75-10eb77f8d0c6 Token:2336412f37fb687f5d51e6e241d59b68c8bc22259cfc6f693fd83967d54f8fe51e3ed79666e0c885a59b1b8b95475c914d4dd4ee9cf1e1e3c42cce937d340492da6b61f6e6fa6eedae3c506c4b3454b71f5cbab5f215e568588c184d15eb0fbb25c9d9a82d055db44e8ea4dcd8cd932e TopicArn:arn:aws:sns:us-east-1:801199994599:test-topic Subject: Message:You have chosen to subscribe to the topic arn:aws:sns:us-east-1:801199994599:test-topic.\nTo confirm the subscription, visit the SubscribeURL included in this message. SubscribeURL:https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-east-1:801199994599:test-topic&Token=2336412f37fb687f5d51e6e241d59b68c8bc22259cfc6f693fd83967d54f8fe51e3ed79666e0c885a59b1b8b95475c914d4dd4ee9cf1e1e3c42cce937d340492da6b61f6e6fa6eedae3c506c4b3454b71f5cbab5f215e568588c184d15eb0fbb25c9d9a82d055db44e8ea4dcd8cd932e Timestamp:2017-05-18T21:00:39.449Z SignatureVersion:1 Signature:TRDFkOiXN9BpKy1p7b+jSZa3d/gvDozddOpnQ9yRkmTFX4OoNCLo4xHMXjmybB6ElKX3swi3Z51eg540ACOLpryyGLtubtV5J2BaLb/fxDuqJjNDMy1clmUheZKVQ1nLDTmcdckS5bqSA9V7Sttxpxleqxwkl77+deCWoZ9L+C7mU19SaY8B7Iau47/AS99ghbU8VU45/gcNsrDuAi2fhT2dF4RFPyfM3E0MA6MRwqwj9oxf+3dIAZDIP3spdcTtFKA9fHAOImnTPMEo47hzSilEMWz4MmfAPHA0Hnh6T6IIFdy75RhNhrA/mF3Kmfj5Xw0HJK6n2LH8cRZv7HMJEg== SigningCertURL:https://sns.us-east-1.amazonaws.com/SimpleNotificationService-b95095beb82e8f6a046b3aafc7f4149a.pem UnsubscribeURL: MessageAttributes:map[]}
*/
	var ma map[string]MsgAttr
	confirmation := &SNSMessage{
		Type: "SubscriptionConfirmation",
		MessageId: "76fa7c37-969c-458a-9d75-10eb77f8d0c6",
		Token: "2336412f37fb687f5d51e6e241d59b68c8bc22259cfc6f693fd83967d54f8fe51e3ed79666e0c885a59b1b8b95475c914d4dd4ee9cf1e1e3c42cce937d340492da6b61f6e6fa6eedae3c506c4b3454b71f5cbab5f215e568588c184d15eb0fbb25c9d9a82d055db44e8ea4dcd8cd932e",
		TopicArn: "arn:aws:sns:us-east-1:801199994599:test-topic",
		Subject: "",
		Message: "You have chosen to subscribe to the topic arn:aws:sns:us-east-1:801199994599:test-topic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
		SubscribeURL: "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-east-1:801199994599:test-topic&Token=2336412f37fb687f5d51e6e241d59b68c8bc22259cfc6f693fd83967d54f8fe51e3ed79666e0c885a59b1b8b95475c914d4dd4ee9cf1e1e3c42cce937d340492da6b61f6e6fa6eedae3c506c4b3454b71f5cbab5f215e568588c184d15eb0fbb25c9d9a82d055db44e8ea4dcd8cd932e",
		Timestamp: "2017-05-18T21:00:39.449Z",
		SignatureVersion: "1",
		Signature: "TRDFkOiXN9BpKy1p7b+jSZa3d/gvDozddOpnQ9yRkmTFX4OoNCLo4xHMXjmybB6ElKX3swi3Z51eg540ACOLpryyGLtubtV5J2BaLb/fxDuqJjNDMy1clmUheZKVQ1nLDTmcdckS5bqSA9V7Sttxpxleqxwkl77+deCWoZ9L+C7mU19SaY8B7Iau47/AS99ghbU8VU45/gcNsrDuAi2fhT2dF4RFPyfM3E0MA6MRwqwj9oxf+3dIAZDIP3spdcTtFKA9fHAOImnTPMEo47hzSilEMWz4MmfAPHA0Hnh6T6IIFdy75RhNhrA/mF3Kmfj5Xw0HJK6n2LH8cRZv7HMJEg==",
		SigningCertURL: "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-b95095beb82e8f6a046b3aafc7f4149a.pem",
		UnsubscribeURL: "",
		MessageAttributes: ma,
	}
	
	return notification, confirmation
}

func testCreateCerficate() (privkey *rsa.PrivateKey, pemkey []byte, err error) {
	template := &x509.Certificate {
		SignatureAlgorithm: x509.SHA256WithRSA,
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
	h := sha1.Sum([]byte( formatSignature(snsMsg) ))
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
	snsMsgBad_noti  := *snsMsg_noti
	snsMsgBad_noti.Subject = "No more room for Jello"
	
	snsMsgGood_conf := *snsMsg_conf
	snsMsgBad_conf  := *snsMsg_conf
	snsMsgBad_conf.Message = "bad confirmation message"
	
	msgs = map[string]*SNSMessage{
		"noti-good": &snsMsgGood_noti,
		"noti-bad": &snsMsgBad_noti,
		"conf-good": &snsMsgGood_conf,
		"conf-bad": &snsMsgBad_conf,
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
	_, errBad  := base64Decode(snsMsg["noti-bad"])
	
	assert.Nil(errGood)
	assert.Nil(errBad)
	
	_, errGood = base64Decode(snsMsg["conf-good"])
	_, errBad  = base64Decode(snsMsg["conf-bad"])
	
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
	_, errBad  := v.getPemFile(snsMsg["noti-bad"].SigningCertURL)
	
	assert.Nil(errGood)
	assert.Nil(errBad)
	
	_, errGood = v.getPemFile(snsMsg["conf-good"].SigningCertURL)
	_, errBad  = v.getPemFile(snsMsg["conf-bad"].SigningCertURL)
	
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
	pemFromBad, err  := v.getPemFile(snsMsg["noti-bad"].SigningCertURL)
	assert.Nil(err)

	_, errGood := getCerticate(pemFromGood)
	_, errBad  := getCerticate(pemFromBad)
	
	assert.Nil(errGood)
	assert.Nil(errBad)
	
	pemFromGood, err = v.getPemFile(snsMsg["conf-good"].SigningCertURL)
	assert.Nil(err)
	pemFromBad, err  = v.getPemFile(snsMsg["conf-bad"].SigningCertURL)
	assert.Nil(err)

	_, errGood = getCerticate(pemFromGood)
	_, errBad  = getCerticate(pemFromBad)
	
	assert.Nil(errGood)
	assert.Nil(errBad)
}

func Test_generateSignature(t *testing.T) {
	assert := assert.New(t)
	
	snsMsg_noti, snsMsg_conf := testSNSMessage("127.0.0.1")
	h1 := generateSignature(snsMsg_noti)
	h2 := generateSignature(snsMsg_conf)
	
	assert.NotNil(h1)
	assert.NotNil(h2)
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
