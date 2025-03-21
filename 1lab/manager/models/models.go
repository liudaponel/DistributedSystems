package models

type RequestStatus string

const (
	StatusNew          RequestStatus = "NEW"
	StatusInProgress   RequestStatus = "IN_PROGRESS"
	StatusReady        RequestStatus = "READY"
	StatusError        RequestStatus = "ERROR"
	StatusPartialReady RequestStatus = "PARTIAL_READY"
)

type StartRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"maxLength"`
}

type StartResponse struct {
	RequestId string `json:"requestId"`
}

type RequestInfo struct {
	Status   RequestStatus
	Data     []string
	Total    int
	Received int
}

type StatusResponse struct {
	Status   RequestStatus `json:"status"`
	Progress int           `json:"progress"`
	Data     *[]string     `json:"data"`
}

type WorkerRequest struct {
	RequestId  string `json:"requestId"`
	PartNumber int    `json:"partNumber"`
	PartCount  int    `json:"partCount"`
	Hash       string `json:"hash"`
	MaxLength  int    `json:"maxLength"`
}

type WorkerResponse struct {
	RequestId  string   `json:"requestId"`
	Words      []string `json:"words"`
	PartNumber int      `json:"partNumber"`
}

type ProgressResponse struct {
	RequestId  string `json:"requestId"`
	PartNumber int    `json:"partNumber"`
	Progress   float64
}
