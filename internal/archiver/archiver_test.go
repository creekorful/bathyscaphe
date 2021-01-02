package archiver

import (
	"github.com/creekorful/trandoshan/internal/archiver/storage_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestHandleNewResourceEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	storageMock := storage_mock.NewMockStorage(mockCtrl)

	tn := time.Now()

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.NewIndexEvent{}).
		SetArg(1, event.NewIndexEvent{
			URL:     "https://example.onion",
			Body:    "Hello, world",
			Headers: map[string]string{"Server": "Traefik", "Content-Type": "application/html"},
			Time:    tn,
		}).Return(nil)

	storageMock.EXPECT().Store("https://example.onion", tn, []byte("Server: Traefik\r\nContent-Type: application/html\r\n\r\nHello, world")).Return(nil)

	s := State{storage: storageMock}
	if err := s.handleNewIndexEvent(subscriberMock, msg); err != nil {
		t.Fail()
	}
}

func TestFormatResource(t *testing.T) {
	evt := &event.NewIndexEvent{
		URL:     "https://google.com",
		Body:    "Hello, world",
		Headers: map[string]string{"Server": "Traefik", "Content-Type": "text/html"},
		Time:    time.Now(),
	}

	res, err := formatResource(evt)
	if err != nil {
		t.FailNow()
	}

	if string(res) != "Server: Traefik\r\nContent-Type: text/html\r\n\r\nHello, world" {
		t.Fail()
	}
}
