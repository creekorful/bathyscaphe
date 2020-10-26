package api

import (
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/api/database"
	"github.com/creekorful/trandoshan/internal/api/database_mock"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/messaging_mock"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestSearchResources(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := &database.ResSearchParams{Keyword: "example"}

	dbMock := database_mock.NewMockDatabase(mockCtrl)

	dbMock.EXPECT().CountResources(params).Return(int64(150), nil)
	dbMock.EXPECT().SearchResources(params).Return([]database.ResourceIdx{
		{
			URL:   "example-1.onion",
			Body:  "Example 1",
			Title: "Example 1",
			Time:  time.Time{},
		},
		{
			URL:   "example-2.onion",
			Body:  "Example 2",
			Title: "Example 2",
			Time:  time.Time{},
		},
	}, nil)

	s := svc{db: dbMock}

	res, count, err := s.searchResources(params)
	if err != nil {
		t.FailNow()
	}
	if count != 150 {
		t.Error()
	}
	if len(res) != 2 {
		t.Error()
	}
}

func TestAddResource(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dbMock := database_mock.NewMockDatabase(mockCtrl)

	dbMock.EXPECT().AddResource(database.ResourceIdx{
		URL:   "https://example.onion",
		Body:  "TheBody",
		Title: "Example",
		Time:  time.Time{},
	})

	s := svc{db: dbMock}

	res, err := s.addResource(api.ResourceDto{
		URL:   "https://example.onion",
		Body:  "TheBody",
		Title: "Example",
		Time:  time.Time{},
	})
	if err != nil {
		t.FailNow()
	}

	if res.URL != "https://example.onion" {
		t.FailNow()
	}
	if res.Body != "TheBody" {
		t.FailNow()
	}
	if res.Title != "Example" {
		t.FailNow()
	}
	if !res.Time.IsZero() {
		t.FailNow()
	}
}

func TestScheduleURL(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pubMock := messaging_mock.NewMockPublisher(mockCtrl)

	s := svc{pub: pubMock}

	pubMock.EXPECT().PublishMsg(&messaging.URLFoundMsg{URL: "https://example.onion"})

	if err := s.scheduleURL("https://example.onion"); err != nil {
		t.FailNow()
	}
}

func TestAuthenticateInvalidCredentials(t *testing.T) {
	s := svc{}

	if _, err := s.authenticate(api.CredentialsDto{}); err == nil {
		t.FailNow()
	}
}

func TestAuthenticateWrongCredentials(t *testing.T) {
	s := svc{users: map[string][]byte{"creekorful": []byte("")}}

	if _, err := s.authenticate(api.CredentialsDto{Username: "johndoe", Password: "test"}); err == nil {
		t.FailNow()
	}
	if _, err := s.authenticate(api.CredentialsDto{Username: "creekorful", Password: "tes"}); err == nil {
		t.FailNow()
	}
}

func TestAuthenticate(t *testing.T) {
	s := svc{
		users: map[string][]byte{
			"creekorful": []byte("$2a$10$aLX2t8JsTOoy9iRLBNm.RuPMmcA8MCXijuzhLvUwUbSlh.C/D2eLm")},
		signingKey: []byte("secret"),
	}

	tokenStr, err := s.authenticate(api.CredentialsDto{Username: "creekorful", Password: "test"})
	if err != nil {
		t.FailNow()
	}

	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if token.Header["alg"] != jwt.SigningMethodHS256.Alg() {
		t.Errorf("Invalid alg: %s", token.Header["alg"])
	}
	if claims["username"] != "creekorful" {
		t.Errorf("Invalid username: %s", claims["username"])
	}
}
