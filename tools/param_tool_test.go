package tools

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(method, path, nil)
	c.Request = req
	return c, w
}

func TestParseRequiredParam_Present(t *testing.T) {
	c, _ := newTestContext("GET", "/")
	c.Params = gin.Params{{Key: "id", Value: "abc-123"}}

	val, ok := ParseRequiredParam(c, "id", "T", "EP01", "invalid id")
	assert.True(t, ok)
	assert.Equal(t, "abc-123", val)
}

func TestParseRequiredParam_Missing(t *testing.T) {
	c, w := newTestContext("GET", "/")
	c.Params = gin.Params{}

	val, ok := ParseRequiredParam(c, "id", "T", "EP01", "invalid id")
	assert.False(t, ok)
	assert.Empty(t, val)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseIntParam_Valid(t *testing.T) {
	c, _ := newTestContext("GET", "/")
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	val, ok := ParseIntParam(c, "id", "T", "EP01", "invalid id")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestParseIntParam_NotInt(t *testing.T) {
	c, w := newTestContext("GET", "/")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	_, ok := ParseIntParam(c, "id", "T", "EP01", "invalid id")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseIntParam_Missing(t *testing.T) {
	c, w := newTestContext("GET", "/")
	c.Params = gin.Params{}

	_, ok := ParseIntParam(c, "id", "T", "EP01", "invalid id")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseQueryInt_Default(t *testing.T) {
	c, _ := newTestContext("GET", "/")

	val, ok := ParseQueryInt(c, "limit", 20, 1, 100, "T", "EP", "invalid")
	require.True(t, ok)
	assert.Equal(t, 20, val)
}

func TestParseQueryInt_Provided(t *testing.T) {
	c, _ := newTestContext("GET", "/?limit=50")
	c.Request.URL.RawQuery = "limit=50"

	val, ok := ParseQueryInt(c, "limit", 20, 1, 100, "T", "EP", "invalid")
	require.True(t, ok)
	assert.Equal(t, 50, val)
}

func TestParseQueryInt_OutOfRange(t *testing.T) {
	c, w := newTestContext("GET", "/?limit=200")
	c.Request.URL.RawQuery = "limit=200"

	_, ok := ParseQueryInt(c, "limit", 20, 1, 100, "T", "EP", "invalid")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseQueryPageLimit_Defaults(t *testing.T) {
	c, _ := newTestContext("GET", "/")

	page, limit, ok := ParseQueryPageLimit(c, "T", "EP")
	require.True(t, ok)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, limit)
}

func TestParseQueryEnum_Valid(t *testing.T) {
	c, _ := newTestContext("GET", "/?sort=asc")
	c.Request.URL.RawQuery = "sort=asc"

	val, ok := ParseQueryEnum(c, "sort", "asc", []string{"asc", "desc"}, "T", "EP")
	require.True(t, ok)
	assert.Equal(t, "asc", val)
}

func TestParseQueryEnum_Invalid(t *testing.T) {
	c, w := newTestContext("GET", "/?sort=random")
	c.Request.URL.RawQuery = "sort=random"

	_, ok := ParseQueryEnum(c, "sort", "asc", []string{"asc", "desc"}, "T", "EP")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseQueryEnum_Default(t *testing.T) {
	c, _ := newTestContext("GET", "/")

	val, ok := ParseQueryEnum(c, "sort", "asc", []string{"asc", "desc"}, "T", "EP")
	require.True(t, ok)
	assert.Equal(t, "asc", val)
}
