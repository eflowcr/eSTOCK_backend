package requests

type CompleteTasks struct {
	TaskType       string `json:"task_type" validate:"required,max=50"`
	CompletionTime int    `json:"completion_time" validate:"gte=0"`
	Accuracy       *int   `json:"accuracy" validate:"omitempty,gte=0,lte=100"`
}
