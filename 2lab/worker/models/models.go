package models

type TaskRequest struct {
	RequestId  string `json:"requestId"`
	PartNumber int    `json:"partNumber"`
	PartCount  int    `json:"partCount"`
	Hash       string `json:"hash"`
	MaxLength  int    `json:"maxLength"`
}

type TaskResponse struct {
	RequestId  string   `json:"requestId"`
	Words      []string `json:"words"`
	PartNumber int      `json:"partNumber"`
}

type TaskProgress struct {
	Progress   float64
	PartNumber int
}

type ProgressResponse struct {
	RequestId  string `json:"requestId"`
	PartNumber int    `json:"partNumber"`
	Progress   float64
}
