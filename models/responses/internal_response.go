package responses

type InternalResponse struct {
	Error   error
	Message string
	Handled bool
}

func InternalErrorResponse(err error, message string, handled bool) InternalResponse {
	return InternalResponse{
		Error:   err,
		Message: message,
		Handled: handled,
	}
}
