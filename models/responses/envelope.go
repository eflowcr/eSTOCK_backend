package responses

type Envelope struct {
	TransactionType string `json:"transaction_type"`
	Encrypted       bool   `json:"encrypted"`
	EncryptionType  string `json:"encryption_type"`
}
