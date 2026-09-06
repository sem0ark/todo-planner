package main

import (
	"net/http"
	"strconv"
	"strings"
)

func (api *API) categoriesHandler(responseWriter http.ResponseWriter, request *http.Request) {
	api.categoriesRouter(responseWriter, request)
}

func (api *API) categoriesRouter(responseWriter http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/categories")
	if path == "" || path == "/" {
		switch request.Method {
		case http.MethodGet:
			api.getCategoriesHandler(responseWriter, request)
		case http.MethodPost:
			api.createCategoryHandler(responseWriter, request)
		default:
			methodNotAllowed(responseWriter)
		}
		return
	}
	id, err := strconv.Atoi(strings.Trim(path, "/"))
	if err != nil {
		http.Error(responseWriter, "invalid category ID", http.StatusBadRequest)
		return
	}
	switch request.Method {
	case http.MethodPut:
		api.updateCategoryHandler(responseWriter, request, id)
	case http.MethodDelete:
		api.deleteCategoryHandler(responseWriter, request, id)
	default:
		methodNotAllowed(responseWriter)
	}
}
