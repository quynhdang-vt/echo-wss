package models

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetWorkRequestToMessageAndBack(t *testing.T) {
	p := GetWorkRequest{
		Name: "name1",
		ID:   "id1",
		TTL:  1000,
	}

	msgType, bArr, err := LocalInterfaceToBytes(p)
	assert.Contains(t, string(bArr), getWorkRequest)
	t.Logf("msgType=%d, %s, err=%v", msgType, string(bArr), err)
	p2, err := BytesToGetWorkRequest(msgType, bArr)
	assert.Nil(t, err)
	assert.NotNil(t, p2)
	t.Logf("p2 = %s, err =%v", ToString(p2), err)
	assert.Equal(t, p.Name, p2.Name)
	assert.Equal(t, p.ID, p2.ID)
	assert.Equal(t, p.TTL, p2.TTL)

	p3, err := BytesToGetWorkResponse(msgType, bArr)
	assert.Nil(t, p3)
	assert.NotNil (t, err)
	t.Logf("Should get err, %v", err)
}

func TestGetWorkResponseToMessageAndBack(t *testing.T) {
	p := GetWorkResponse{
		Name: "name1",
		ID:   "id1",
		WorkItem: "workItem1",
	}

	msgType, bArr, err := LocalInterfaceToBytes(p)
	assert.Contains(t, string(bArr), getWorkResponse)
	t.Logf("msgType=%d, %s, err=%v", msgType, string(bArr), err)
	p2, err := BytesToGetWorkResponse(msgType, bArr)
	assert.Nil(t, err)
	assert.NotNil(t, p2)
	t.Logf("p2 = %s, err =%v", ToString(p2), err)
	assert.Equal(t, p.Name, p2.Name)
	assert.Equal(t, p.ID, p2.ID)
	assert.Equal(t, p.WorkItem, p2.WorkItem)

	p3, err := BytesToGetWorkRequest(msgType, bArr)
	assert.Nil(t, p3)
	assert.NotNil (t, err)
	t.Logf("Should get err, %v", err)
}