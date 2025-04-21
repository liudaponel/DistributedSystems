package models

type RequestStatus string

const (
	StatusNew          RequestStatus = "NEW"
	StatusInProgress   RequestStatus = "IN_PROGRESS"
	StatusReady        RequestStatus = "READY"
	StatusError        RequestStatus = "ERROR"
	StatusPartialReady RequestStatus = "PARTIAL_READY"
	StatusTimeout      RequestStatus = "TIME_OUT"
)

type StartRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"maxLength"`
}

type StartResponse struct {
	RequestId string `json:"requestId"`
}

type RequestInfoMongo struct {
	ID            string        `bson:"_id"`
	Hash          string        `bson:"hash"`
	MaxLength     int           `bson:"max_length"`
	Status        RequestStatus `bson:"status"`
	TotalParts    int           `bson:"total_parts"`
	ReceivedParts int           `bson:"received_parts"`
	Data          []string      `bson:"data,omitempty"`
	ErrorMsg      string        `bson:"error_msg,omitempty"`
}

type PendingTask struct {
	ID        string `bson:"_id"` // Уникальный ID для этой записи
	RequestID string `bson:"request_id"`
	TaskBody  []byte `bson:"task_body"` // Сериализованный WorkerRequest
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
