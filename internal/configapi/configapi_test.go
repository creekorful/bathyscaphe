package configapi

import (
	"github.com/darkspot-org/bathyscaphe/internal/cache"
	"github.com/darkspot-org/bathyscaphe/internal/cache_mock"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"github.com/darkspot-org/bathyscaphe/internal/event_mock"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/darkspot-org/bathyscaphe/internal/process_mock"
	"github.com/darkspot-org/bathyscaphe/internal/test"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestState_Name(t *testing.T) {
	s := State{}
	if s.Name() != "configapi" {
		t.Fail()
	}
}

func TestState_Features(t *testing.T) {
	s := State{}
	test.CheckProcessFeatures(t, &s, []process.Feature{process.EventFeature, process.CacheFeature})
}

func TestState_CustomFlags(t *testing.T) {
	s := State{}
	test.CheckProcessCustomFlags(t, &s, []string{"default-value"})
}

func TestState_Initialize(t *testing.T) {
	test.CheckInitialize(t, &State{}, func(p *process_mock.MockProviderMockRecorder) {
		p.Cache("configuration")
		p.Publisher()
		p.GetStrValues("default-value")
	})
}

func TestState_Subscribers(t *testing.T) {
	s := State{}
	test.CheckProcessSubscribers(t, &s, nil)
}

func TestGetConfiguration(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	configCacheMock := cache_mock.NewMockCache(mockCtrl)
	configCacheMock.EXPECT().GetBytes("hello").Return([]byte("{\"ttl\": \"10s\"}"), nil)

	req := httptest.NewRequest(http.MethodGet, "/config/hello", nil)
	req = mux.SetURLVars(req, map[string]string{"key": "hello"})

	rec := httptest.NewRecorder()

	s := State{configCache: configCacheMock}
	s.getConfiguration(rec, req)

	if rec.Code != http.StatusOK {
		t.Fail()
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fail()
	}

	b, err := ioutil.ReadAll(rec.Body)
	if err != nil {
		t.FailNow()
	}

	if string(b) != "{\"ttl\": \"10s\"}" {
		t.Fail()
	}
}

func TestSetConfiguration(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	configCacheMock := cache_mock.NewMockCache(mockCtrl)
	pubMock := event_mock.NewMockPublisher(mockCtrl)

	configCacheMock.EXPECT().SetBytes("hello", []byte("{\"ttl\": \"10s\"}"), cache.NoTTL).Return(nil)
	pubMock.EXPECT().PublishJSON("config", event.RawMessage{
		Body:    []byte("{\"ttl\": \"10s\"}"),
		Headers: map[string]interface{}{"Config-Key": "hello"},
	}).Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/config/hello", strings.NewReader("{\"ttl\": \"10s\"}"))
	req = mux.SetURLVars(req, map[string]string{"key": "hello"})

	rec := httptest.NewRecorder()

	s := State{configCache: configCacheMock, pub: pubMock}
	s.setConfiguration(rec, req)

	if rec.Code != http.StatusOK {
		t.Fail()
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fail()
	}

	b, err := ioutil.ReadAll(rec.Body)
	if err != nil {
		t.FailNow()
	}

	if string(b) != "{\"ttl\": \"10s\"}" {
		t.Fail()
	}
}
