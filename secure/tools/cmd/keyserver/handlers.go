package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
)

const (
	KeyIDVariableName = "kid"

	KeyIDParameterName = "kid"
	ExpParameterName   = "exp"
	NbfParameterName   = "nbf"
)

// BasicHandler handles all keyserver requests
type BasicHandler struct {
	keyStore    *KeyStore
	errorLogger *log.Logger
}

func (handler *BasicHandler) GetKey(response http.ResponseWriter, request *http.Request) {
	variables := mux.Vars(request)
	keyID := variables[KeyIDVariableName]
	if len(keyID) == 0 {
		handler.errorLogger.Println("No key identifier supplied")
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	key, ok := handler.keyStore.PublicKey(keyID)
	if ok {
		// Should we use application/x-pem-file instead?
		response.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		response.Write(key)
	} else {
		message := fmt.Sprintf("No such key: %s", keyID)
		handler.errorLogger.Println(message)

		response.Header().Set("Content-Type", "application/json;charset=UTF-8")
		response.WriteHeader(http.StatusNotFound)

		response.Write(
			[]byte(fmt.Sprintf(`{"message": "%s"}`, message)),
		)
	}
}

func (handler *BasicHandler) ListKeys(response http.ResponseWriter, request *http.Request) {
	keyIDs := handler.keyStore.KeyIDs()
	response.Header().Set("Content-Type", "application/json;charset=UTF-8")
	response.Write(
		[]byte(fmt.Sprintf(`{"keyIds": [%s]}`, strings.Join(keyIDs, ","))),
	)
}

func (handler *BasicHandler) GenerateJWTFromBody(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		handler.errorLogger.Println(err)
		response.WriteHeader(http.StatusBadRequest)
		return
	}
}
