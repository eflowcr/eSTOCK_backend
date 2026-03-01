package responses

// HTTP status codes for InternalResponse.StatusCode (same values as net/http).
// Controllers use these to choose the right response helper.
const (
	StatusBadRequest          = 400
	StatusNotFound            = 404
	StatusConflict            = 409
	StatusInternalServerError = 500
)

// InternalResponse is returned by services/repos to signal an error or expected outcome.
// When StatusCode is non-zero, the controller should use the matching HTTP status helper
// (e.g. 404 → ResponseNotFound, 409 → ResponseConflict). When 0, legacy behaviour applies
// (Response with 200 if Handled, 400 if not).
type InternalResponse struct {
	Error      error
	Message    string
	Handled    bool
	StatusCode int // optional: 400, 404, 409, 500, etc.; 0 = use Handled for legacy 200/400
}

func InternalErrorResponse(err error, message string, handled bool) InternalResponse {
	return InternalResponse{
		Error:   err,
		Message: message,
		Handled: handled,
	}
}
