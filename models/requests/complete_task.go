package requests

type CompleteTasks struct {
	TaskType       string `json:"task_type"`
	CompletionTime int    `json:"completion_time"`
	Accuracy       *int   `json:"accuracy"`
}
