package model

type ConvertRequest struct {
	RequestID      string `json:"request_id"`
	CollelationKey string `json:"colleration_key"`
	HtmlS3Key      string `json:"html_s3_key"`
	Bucket         string `json:"bucket"`
}

type Job struct {
	Data []byte
	Ack  func()
}
