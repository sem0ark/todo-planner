package main

import (
	"net/http"
	"strconv"
	"strings"
)

func (api *API) templateGroupsHandler(responseWriter http.ResponseWriter, request *http.Request) {
	api.templateGroupsRouter(responseWriter, request)
}

func (api *API) templateGroupsRouter(responseWriter http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/template-groups")
	if path == "" || path == "/" {
		switch request.Method {
		case http.MethodGet:
			api.getTemplateGroupsHandler(responseWriter, request)
		case http.MethodPost:
			api.createTemplateGroupHandler(responseWriter, request)
		default:
			methodNotAllowed(responseWriter)
		}
		return
	}
	id, err := strconv.Atoi(strings.Trim(path, "/"))
	if err != nil {
		http.Error(responseWriter, "invalid template group id", http.StatusBadRequest)
		return
	}
	switch request.Method {
	case http.MethodPut:
		api.updateTemplateGroupHandler(responseWriter, request, id)
	case http.MethodDelete:
		api.deleteTemplateGroupHandler(responseWriter, request, id)
	default:
		methodNotAllowed(responseWriter)
	}
}
