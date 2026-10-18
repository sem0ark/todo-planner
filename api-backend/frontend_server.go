//go:build !frontend

package main

import (
	"errors"
	"net/http"
)

func newFrontendHandler() (http.Handler, error) {
	return nil, errors.New("frontend hosting is not enabled in this build")
}
