package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// KeyHandler handles key-related requests
type KeyHandler struct {
	BasicHandler
}

func (handler *KeyHandler) GetKey(response http.ResponseWriter, request *http.Request) {
	variables := mux.Vars(request)
	keyID := variables[KeyIDVariableName]
	if len(keyID) == 0 {
		handler.httpError(response, http.StatusBadRequest, "No key identifier supplied")
		return
	}

	key, ok := handler.keyStore.PublicKey(keyID)
	if ok {
		// Should we use application/x-pem-file instead?
		response.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		response.Write(key)
	} else {
		message := fmt.Sprintf("No such key: %s", keyID)
		handler.logger.Println(message)

		response.Header().Set("Content-Type", "application/json;charset=UTF-8")
		response.WriteHeader(http.StatusNotFound)

		response.Write(
			[]byte(fmt.Sprintf(`{"message": "%s"}`, message)),
		)
	}
}

func (handler *KeyHandler) ListKeys(response http.ResponseWriter, request *http.Request) {
	keyIDs := handler.keyStore.KeyIDs()
	response.Header().Set("Content-Type", "application/json;charset=UTF-8")
	response.Write(
		[]byte(fmt.Sprintf(`{"keyIds": [%s]}`, strings.Join(keyIDs, ","))),
	)
}
