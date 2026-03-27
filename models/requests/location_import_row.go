package requests

// LocationImportRow represents a single location row sent from the frontend preview table.
type LocationImportRow struct {
	LocationCode string `json:"location_code"`
	Description  string `json:"description"`
	Zone         string `json:"zone"`
	Type         string `json:"type"`
}
