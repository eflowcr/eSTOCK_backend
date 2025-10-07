package database

type Presentations struct {
	PresentationId string `gorm:"presentation_id" json:"presentation_id"`
	Description    string `gorm:"description" json:"description"`
}

func (Presentations) TableName() string {
	return "presentations"
}
