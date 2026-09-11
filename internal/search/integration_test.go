package search_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/search"
	"github.com/THEcanon001/turnero/internal/testutil"
)

func TestSearch_Integration_TrigramMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	// Seed some providers
	testutil.SeedProvider(t, pool, "barberia-juan")
	testutil.SeedProvider(t, pool, "peluqueria-maria")
	testutil.SeedProvider(t, pool, "spa-relax")

	h := search.NewHandler(pool)

	// Search for "barber" — should match "barberia-juan" via ILIKE
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=barber", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var results []search.Result
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&results))
	require.NotEmpty(t, results)
	assert.Contains(t, results[0].Slug, "barberia")
}

func TestSearch_Integration_PartialName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	testutil.SeedProvider(t, pool, "clinica-dental")
	testutil.SeedProvider(t, pool, "clinica-medica")

	h := search.NewHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=clinica", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var results []search.Result
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&results))
	assert.Len(t, results, 2)
}

func TestSearch_Integration_NoMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	testutil.SeedProvider(t, pool, "barberia")

	h := search.NewHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=zzzznothing", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var results []search.Result
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&results))
	assert.Empty(t, results)
}

func TestSearch_Integration_SlugMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	testutil.SeedProvider(t, pool, "unique-slug-xyz")

	h := search.NewHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=unique-slug", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var results []search.Result
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&results))
	require.NotEmpty(t, results)
	assert.Equal(t, "unique-slug-xyz", results[0].Slug)
}

func TestSearch_Integration_MinLength(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	// Ensure pg_trgm extension exists (migrations should handle it)
	_, err := pool.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS pg_trgm")
	require.NoError(t, err)

	testutil.SeedProvider(t, pool, "ab-provider")

	h := search.NewHandler(pool)

	// 2 chars should work
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=ab", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var results []search.Result
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&results))
	assert.NotEmpty(t, results)
}
