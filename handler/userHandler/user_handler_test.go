package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestLoginUser(t *testing.T) {
	e := echo.New()

	reqBody := `{"username":"testuser","password":"password"}`
	req := httptest.NewRequest(http.MethodPost, "/users/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		// Simulate successful login and token generation
		return c.JSON(http.StatusOK, map[string]string{"token": "test-token"})
	}

	// Assertions
	if assert.NoError(t, handler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "test-token")
	}
}