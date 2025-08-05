package responses

type APIResponse struct {
	Envelope Envelope    `json:"envelope"`
	Result   Result      `json:"result"`
	Data     interface{} `json:"data"`
}
