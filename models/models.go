package models

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
)

const (
	unknown = "unknown"
	getWorkRequest = "getWorkRequest"
	getWorkResponse = "getWorkResponse"
)
type Message struct {
	MessageCode string   // so that we can properly deserialize based on the type
	Data        json.RawMessage
}

type GetWorkResponse struct {
	Name     string `json:"name,omitempty"`
	ID       string `json:"id,omitempty`
	WorkItem string `json:"workItem,omitempty"`
}


type GetWorkRequest struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty`
	TTL  int64  `json:"ttl,omitempty"`
}

func ToString(c interface{}) string {
	if c == nil {
		return ""
	}
	s, _ := json.MarshalIndent(c, "", "\t")
	return string(s)
}
func LocalInterfaceToBytes (p interface{}) (int, []byte, error) {
	b, _ := json.Marshal(p)
	msgCode := unknown
	if _, ok := p.(GetWorkRequest); ok {
		msgCode= getWorkRequest
	}
	if _, ok := p.(GetWorkResponse); ok {
		msgCode= getWorkResponse
	}
	msg := Message{
		MessageCode: msgCode,
		Data: b,
	}
	res, err := json.Marshal(msg)
	return websocket.TextMessage, res, err
}

func byteArrToCorrectType(msgType int, b []byte) (interface{}, error) {
	if msgType != websocket.TextMessage {
		return nil, fmt.Errorf("unrecognized %d", msgType)
	}
	var msg = &Message{}
	err := json.Unmarshal(b, msg)
	if err != nil {
		return nil, err
	}

	switch msg.MessageCode {
	case getWorkResponse:
		var res = new(GetWorkResponse)
		err = json.Unmarshal(msg.Data, res)
		return res, err
	case getWorkRequest:
		var res = new(GetWorkRequest)
		err = json.Unmarshal(msg.Data, res)
		return res, err
	}
	return nil, fmt.Errorf("Unrecognized %s", msg.MessageCode)
}

func BytesToGetWorkRequest(msgType int, b []byte) (*GetWorkRequest, error) {
	p, err := byteArrToCorrectType(msgType, b)
	if err != nil {
		return nil, err
	}
	if res, ok := p.(*GetWorkRequest); ok {
		return res, nil
	}
	return nil, fmt.Errorf("Failed to assert to GetWorkRequest")
}
func BytesToGetWorkResponse(msgType int, b []byte) (*GetWorkResponse, error) {
	p, err := byteArrToCorrectType(msgType, b)
	if err != nil {
		return nil, err
	}
	if res, ok := p.(*GetWorkResponse); ok {
		return res, nil
	}
	return nil, fmt.Errorf("Failed to assert to GetWorkResponse")
}