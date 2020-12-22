package crawler

import (
	"bytes"
	"github.com/creekorful/trandoshan/internal/crawler/http_mock"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/messaging_mock"
	"github.com/golang/mock/gomock"
	"strings"
	"testing"
)

func TestCrawlURLForbiddenContentType(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	httpClientMock := http_mock.NewMockClient(mockCtrl)
	url := "https://example.onion"
	allowedContentTypes := []string{"text/plain"}

	httpResponseMock := http_mock.NewMockResponse(mockCtrl)
	httpResponseMock.EXPECT().Headers().Return(map[string]string{"Content-Type": "image/png"})

	httpClientMock.EXPECT().Get(url).Return(httpResponseMock, nil)

	body, headers, err := crawURL(httpClientMock, url, allowedContentTypes)
	if body != "" || headers != nil || err == nil {
		t.Fail()
	}
}

func TestCrawlURLSameContentType(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	httpClientMock := http_mock.NewMockClient(mockCtrl)
	url := "https://example.onion"
	allowedContentTypes := []string{"text/plain"}

	httpResponseMock := http_mock.NewMockResponse(mockCtrl)
	httpResponseMock.EXPECT().Headers().Times(2).Return(map[string]string{"Content-Type": "text/plain"})
	httpResponseMock.EXPECT().Body().Return(strings.NewReader("Hello"))

	httpClientMock.EXPECT().Get(url).Return(httpResponseMock, nil)

	body, headers, err := crawURL(httpClientMock, url, allowedContentTypes)
	if err != nil {
		t.Fail()
	}
	if body != "Hello" {
		t.Fail()
	}
	if len(headers) != 1 {
		t.Fail()
	}
	if headers["Content-Type"] != "text/plain" {
		t.Fail()
	}
}

func TestCrawlURLNoContentTypeFiltering(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	httpClientMock := http_mock.NewMockClient(mockCtrl)
	url := "https://example.onion"
	allowedContentTypes := []string{""}

	httpResponseMock := http_mock.NewMockResponse(mockCtrl)
	httpResponseMock.EXPECT().Headers().Times(2).Return(map[string]string{"Content-Type": "text/plain"})
	httpResponseMock.EXPECT().Body().Return(strings.NewReader("Hello"))

	httpClientMock.EXPECT().Get(url).Return(httpResponseMock, nil)

	body, headers, err := crawURL(httpClientMock, url, allowedContentTypes)
	if err != nil {
		t.Fail()
	}
	if body != "Hello" {
		t.Fail()
	}
	if len(headers) != 1 {
		t.Fail()
	}
	if headers["Content-Type"] != "text/plain" {
		t.Fail()
	}
}

func TestHandleMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := messaging_mock.NewMockSubscriber(mockCtrl)
	httpClientMock := http_mock.NewMockClient(mockCtrl)
	httpResponseMock := http_mock.NewMockResponse(mockCtrl)

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		ReadMsg(msg, &messaging.URLTodoMsg{}).
		SetArg(1, messaging.URLTodoMsg{URL: "https://example.onion/image.png?id=12&test=2"}).
		Return(nil)

	httpResponseMock.EXPECT().Headers().Times(2).Return(map[string]string{"Content-Type": "text/plain", "Server": "Debian"})
	httpResponseMock.EXPECT().Body().Return(strings.NewReader("Hello"))

	httpClientMock.EXPECT().Get("https://example.onion/image.png?id=12&test=2").Return(httpResponseMock, nil)

	subscriberMock.EXPECT().PublishMsg(&messaging.NewResourceMsg{
		URL:     "https://example.onion/image.png?id=12&test=2",
		Body:    "Hello",
		Headers: map[string]string{"Content-Type": "text/plain", "Server": "Debian"},
	}).Return(nil)

	if err := handleMessage(httpClientMock, []string{"text/plain", "text/css"})(subscriberMock, msg); err != nil {
		t.Fail()
	}
}
