package main

import (
	"net/http"
	"strings"
)

func (api *API) daysHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userID, authenticated := getUserID(request.Context())
	if !authenticated {
		http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
		return
	}
	api.routeDays(responseWriter, request, userID)
}

func (api *API) daysRouter(responseWriter http.ResponseWriter, request *http.Request) {
	userID, _ := getUserID(request.Context())
	api.routeDays(responseWriter, request, userID)
}

func (api *API) routeDays(responseWriter http.ResponseWriter, request *http.Request, userID int) {
	path := strings.TrimPrefix(request.URL.Path, "/days")
	if path == "" || path == "/" {
		if request.Method != http.MethodGet {
			methodNotAllowed(responseWriter)
			return
		}
		api.getDays(responseWriter, request, userID)
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 {
		if !isValidCalendarDate(parts[0]) {
			http.Error(responseWriter, "invalid date", http.StatusBadRequest)
			return
		}
		switch request.Method {
		case http.MethodGet:
			api.getDay(responseWriter, request, userID, parts[0])
		case http.MethodPost:
			api.createDay(responseWriter, request, userID, parts[0])
		default:
			methodNotAllowed(responseWriter)
		}
		return
	}

	if len(parts) == 2 && isValidCalendarDate(parts[0]) {
		switch {
		case parts[1] == "events" && request.Method == http.MethodPost:
			api.postDateEvents(responseWriter, request, userID, parts[0])
		case parts[1] == "blocks" && request.Method == http.MethodPut:
			api.putDateBlocks(responseWriter, request, userID, parts[0])
		case parts[1] == "template" && request.Method == http.MethodPut:
			api.putDateTemplate(responseWriter, request, userID, parts[0])
		default:
			notFound(responseWriter)
		}
		return
	}
	notFound(responseWriter)
}
