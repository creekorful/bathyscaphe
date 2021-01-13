package client

import (
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"github.com/darkspot-org/bathyscaphe/internal/event_mock"
	"github.com/golang/mock/gomock"
	"sync"
	"testing"
)

func TestClient(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subMock := event_mock.NewMockSubscriber(mockCtrl)

	// todo find a way to unit test NewConfigClient

	client := &client{
		configAPIURL:       "",
		sub:                subMock,
		mutexes:            map[string]*sync.RWMutex{AllowedMimeTypesKey: {}},
		keys:               []string{AllowedMimeTypesKey},
		forbiddenMimeTypes: []MimeType{},
		forbiddenHostnames: nil,
		refreshDelay:       RefreshDelay{},
	}

	val, err := client.GetAllowedMimeTypes()
	if err != nil {
		t.FailNow()
	}
	if len(val) != 0 {
		t.Fail()
	}

	msg := event.RawMessage{
		Body:    []byte("[{\"content-type\": \"application/json\", \"extensions\": [\"json\"]}]"),
		Headers: map[string]interface{}{"Config-Key": AllowedMimeTypesKey},
	}

	if err := client.handleConfigEvent(subMock, msg); err != nil {
		t.FailNow()
	}

	val, err = client.GetAllowedMimeTypes()
	if err != nil {
		t.FailNow()
	}

	if len(val) != 1 {
		t.FailNow()
	}

	if val[0].ContentType != "application/json" {
		t.Fail()
	}
	if len(val[0].Extensions) != 1 {
		t.FailNow()
	}
	if val[0].Extensions[0] != "json" {
		t.Fail()
	}

}
