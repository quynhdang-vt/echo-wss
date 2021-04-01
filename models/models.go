package models

import (
  "encoding/json"
)

type Message struct {
  MessageCode string
  data json.RawMessage
}

type GetWorkResponse struct {
  Name string `json:"name,omitempty"`
  WorkItem string `json:"workItem"`
}

type GetWorkRequest struct {
  Name string `json:"name,omitempty"`
  TTL int64 `json:"ttl,omitempty"`
}