package main

import (
	"net/http"
	"strconv"
	"strings"
)

func (api *API) dayTemplatesHandler(responseWriter http.ResponseWriter, request *http.Request) {
	api.templatesRouter(responseWriter, request)
}

func (api *API) templatesRouter(responseWriter http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/templates")
	if path == "" || path == "/" {
		switch request.Method {
		case http.MethodGet:
			api.getDayTemplatesHandler(responseWriter, request)
		case http.MethodPost:
			api.createDayTemplateHandler(responseWriter, request)
		default:
			methodNotAllowed(responseWriter)
		}
		return
	}
	id, err := strconv.Atoi(strings.Trim(path, "/"))
	if err != nil {
		http.Error(responseWriter, "invalid template id", http.StatusBadRequest)
		return
	}
	switch request.Method {
	case http.MethodPut:
		api.updateDayTemplateHandler(responseWriter, request, id)
	case http.MethodDelete:
		api.deleteDayTemplateHandler(responseWriter, request, id)
	default:
		methodNotAllowed(responseWriter)
	}
}
