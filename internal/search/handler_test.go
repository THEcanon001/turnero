package search_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/search"
)

func TestSearch_EmptyQuery(t *testing.T) {
	h := search.NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/search", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "[]")
}

func TestSearch_ShortQuery(t *testing.T) {
	h := search.NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=a", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "[]")
}
