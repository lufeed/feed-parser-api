package types

import "net/http"

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (ar *APIResponse) StatusCode() int {
	if ar.Code == 0 {
		return http.StatusInternalServerError
	}
	return ar.Code
}
