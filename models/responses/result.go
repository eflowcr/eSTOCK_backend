package responses

type Result struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	EndpointCode string `json:"endpoint_code"`
}
