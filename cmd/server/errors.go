package main

import (
	"fmt"
	"net/http"
)

type srvError struct {
	Code     int
	Message  string
	LogError error
}

func (e *srvError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%d - %s: %s", e.Code, http.StatusText(e.Code), e.Message)
	}
	return fmt.Sprintf("%d - %s", e.Code, http.StatusText(e.Code))
}

func newHTTPError(code int, message string) error {
	return &srvError{code, message, nil}
}

func newHTTPErrorLog(code int, message string, err error) error {
	return &srvError{code, message, err}
}
