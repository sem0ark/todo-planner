package main

import "net/http"

func methodNotAllowed(responseWriter http.ResponseWriter) {
	http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
}

func notFound(responseWriter http.ResponseWriter) {
	http.Error(responseWriter, "not found", http.StatusNotFound)
}
