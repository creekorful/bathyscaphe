package service

import (
	"github.com/creekorful/trandoshan/internal/configapi/database_mock"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestService_Get(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dbMock := database_mock.NewMockDatabase(mockCtrl)
	dbMock.EXPECT().Get("test").Return([]byte("hello"), nil)

	s := service{
		db: dbMock,
	}

	val, err := s.Get("test")
	if err != nil {
		t.FailNow()
	}
	if string(val) != "hello" {
		t.Fail()
	}
}

func TestService_Set(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dbMock := database_mock.NewMockDatabase(mockCtrl)
	pubMock := event_mock.NewMockPublisher(mockCtrl)

	dbMock.EXPECT().Set("test-key", []byte("hello")).Return(nil)
	pubMock.EXPECT().PublishJSON("config.test-key", []byte("hello")).Return(nil)

	s := service{
		db:  dbMock,
		pub: pubMock,
	}

	if err := s.Set("test-key", []byte("hello")); err != nil {
		t.Fail()
	}
}
