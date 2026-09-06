package main

import "net/http"

// protectedHandler applies authentication once at the route boundary. Endpoint
// handlers can focus on decoding input, invoking repositories, and writing responses.
func (api *API) protectedHandler(handler http.HandlerFunc) http.HandlerFunc {
	return api.authMiddleware(handler)
}

func methodNotAllowed(responseWriter http.ResponseWriter) {
	http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
}

func notFound(responseWriter http.ResponseWriter) {
	http.Error(responseWriter, "not found", http.StatusNotFound)
}
