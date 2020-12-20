package rest

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewSearchParams(t *testing.T) {
	e := echo.New()

	startDate := time.Now()
	target := fmt.Sprintf("/resources?with-body=true&pagination-page=1&keyword=keyword&url=dXJs&start-date=%s", startDate.Format(time.RFC3339))

	req := httptest.NewRequest(http.MethodPost, target, nil)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)

	params, err := newSearchParams(c)
	if err != nil {
		t.Errorf("error while parsing search params: %s", err)
		t.FailNow()
	}

	if !params.WithBody {
		t.Errorf("wrong withBody: %v", params.WithBody)
	}
	if params.PageSize != 50 {
		t.Errorf("wrong pagination-size: %d", params.PageSize)
	}
	if params.PageNumber != 1 {
		t.Errorf("wrong pagination-page: %d", params.PageNumber)
	}
	if params.Keyword != "keyword" {
		t.Errorf("wrong keyword: %s", params.Keyword)
	}
	if params.StartDate.Year() != startDate.Year() {
		t.Errorf("wrong start-date (year)")
	}
	if params.StartDate.Month() != startDate.Month() {
		t.Errorf("wrong start-date (month)")
	}
	if params.StartDate.Day() != startDate.Day() {
		t.Errorf("wrong start-date (day)")
	}
	if params.StartDate.Hour() != startDate.Hour() {
		t.Errorf("wrong start-date (hour)")
	}
	if params.StartDate.Minute() != startDate.Minute() {
		t.Errorf("wrong start-date (minute)")
	}
	if params.StartDate.Second() != startDate.Second() {
		t.Errorf("wrong start-date (second)")
	}
	if params.URL != "url" {
		t.Errorf("wrong url: %s", params.URL)
	}
}
