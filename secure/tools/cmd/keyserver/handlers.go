package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
)

const (
	KeyIdVariableName = "keyId"
)

// KeyHandler handles serving up public keys from a key store
type KeyHandler struct {
	keyStore    *KeyStore
	errorLogger *log.Logger
}

func (kh *KeyHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	variables := mux.Vars(request)
	keyId := variables[KeyIdVariableName]
	if len(keyId) == 0 {
		kh.errorLogger.Println("No key identifier supplied")
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	key, ok := kh.keyStore.PublicKey(keyId)
	if ok {
		// Should we use application/x-pem-file instead?
		response.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		response.Write(key)
	} else {
		message := fmt.Sprintf("No such key: %s", keyId)
		kh.errorLogger.Println(message)

		response.Header().Set("Content-Type", "application/json;charset=UTF-8")
		response.WriteHeader(http.StatusNotFound)

		response.Write(
			[]byte(fmt.Sprintf(`{"message": "%s"}`, message)),
		)
	}
}

type ListKeysHandler struct {
	keyStore    *KeyStore
	errorLogger *log.Logger
}

func (lkh *ListKeysHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	keyIDs := lkh.keyStore.KeyIDs()
	response.Header().Set("Content-Type", "application/json;charset=UTF-8")
	response.Write(
		[]byte(fmt.Sprintf(`{"keyIds": [%s]}`, strings.Join(keyIDs, ","))),
	)
}
