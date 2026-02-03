package model

import "time"

type LogResponse struct {
	Logs       []Log  `json:"logs"`
	NextCursor string `json:"nextCursor"`
	HasMore    bool   `json:"hasMore"`
	Count      int    `json:"count"`
}

type Log struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Level      string    `json:"level"`
	Category   string    `json:"category"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	Source     string    `json:"source"`
	RequestID  string    `json:"requestId,omitempty"`
	HTTPStatus string    `json:"httpStatus,omitempty"`
	IPAddress  string    `json:"ipAddress,omitempty"`
	UserAgent  string    `json:"userAgent,omitempty"`
}
