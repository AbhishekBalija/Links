package app

type errorPayload struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func errorResponse(code, message string) errorPayload {
	return errorPayload{Error: apiError{Code: code, Message: message}}
}
