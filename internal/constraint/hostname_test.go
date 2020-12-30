package constraint

import (
	"github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/configapi/client_mock"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestCheckHostnameAllowed(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	configClientMock := client_mock.NewMockClient(mockCtrl)

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{
		{Hostname: "google.onion"},
	}, nil)
	if allowed, err := CheckHostnameAllowed(configClientMock, "https://google.onion"); allowed || err != nil {
		t.Fail()
	}

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{
		{Hostname: "google.onion"},
	}, nil)
	if allowed, err := CheckHostnameAllowed(configClientMock, "https://google2.onion"); !allowed || err != nil {
		t.Fail()
	}
}
