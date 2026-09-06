package main

import "net/http"

func (api *API) settingsRouter(responseWriter http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		api.getSettingsHandler(responseWriter, request)
	case http.MethodPut:
		api.putSettingsHandler(responseWriter, request)
	default:
		methodNotAllowed(responseWriter)
	}
}

func (api *API) initRouter(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		methodNotAllowed(responseWriter)
		return
	}
	api.initHandler(responseWriter, request)
}

func (api *API) accountRouter(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodDelete {
		methodNotAllowed(responseWriter)
		return
	}
	api.deleteAccountHandler(responseWriter, request)
}

func (api *API) devicesRouter(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		methodNotAllowed(responseWriter)
		return
	}
	api.registerDeviceHandler(responseWriter, request)
}
