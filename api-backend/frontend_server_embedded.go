//go:build frontend

package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// frontendFiles is populated by the build target before a packaged build.
//
//go:embed frontend
var frontendFiles embed.FS

func newFrontendHandler() (http.Handler, error) {
	distribution, err := fs.Sub(frontendFiles, "frontend/dist")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(distribution))

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		path := strings.TrimPrefix(request.URL.Path, "/todo-planner/")
		if path == "" || path == "." {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		request.URL.Path = path
		fileServer.ServeHTTP(response, request)
	}), nil
}
