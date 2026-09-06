package main

import (
	"net/http"
	"strings"
)

func (api *API) scheduleHandler(responseWriter http.ResponseWriter, request *http.Request) {
	api.scheduleRouter(responseWriter, request)
}

func (api *API) scheduleRouter(responseWriter http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/schedule")
	switch {
	case path == "" || path == "/":
		if request.Method != http.MethodGet {
			methodNotAllowed(responseWriter)
			return
		}
		api.getScheduleHandler(responseWriter, request)
	case path == "/weekly":
		if request.Method != http.MethodPut {
			methodNotAllowed(responseWriter)
			return
		}
		api.putWeeklyScheduleHandler(responseWriter, request)
	case strings.HasPrefix(path, "/overrides/"):
		date := strings.TrimPrefix(path, "/overrides/")
		if request.Method == http.MethodPut {
			api.putScheduleOverrideHandler(responseWriter, request, date)
		} else if request.Method == http.MethodDelete {
			api.deleteScheduleOverrideHandler(responseWriter, request, date)
		} else {
			methodNotAllowed(responseWriter)
		}
	default:
		notFound(responseWriter)
	}
}
