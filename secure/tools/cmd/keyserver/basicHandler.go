package main

import (
	"log"
	"net/http"
)

// BasicHandler provides the common behaviors needed by all keyserver handlers
type BasicHandler struct {
	keyStore    *KeyStore
	errorLogger *log.Logger
}

func (handler *BasicHandler) httpError(response http.ResponseWriter, statusCode int, message string) {
	handler.errorLogger.Println(message)
	response.WriteHeader(statusCode)
}
