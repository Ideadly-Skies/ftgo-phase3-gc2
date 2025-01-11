package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"p3/gc2/pb"
)

type MockLibraryServiceClient struct {
	mock.Mock
}

func (m *MockLibraryServiceClient) ReturnBook(ctx context.Context, req *pb.ReturnBookRequest) (*pb.ReturnBookResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.ReturnBookResponse), args.Error(1)
}

func (m *MockLibraryServiceClient) BorrowBook(ctx context.Context, req *pb.BorrowBookRequest) (*pb.BorrowBookResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.BorrowBookResponse), args.Error(1)
}

// unittest for borrowing book 
func TestBorrowBook(t *testing.T) {
	e := echo.New()

	// Mock the JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "test-user-id",
	})
	tokenString, _ := token.SignedString([]byte("12345"))

	// Mock LibraryServiceClient
	mockClient := new(MockLibraryServiceClient)
	mockClient.On("BorrowBook", mock.Anything, &pb.BorrowBookRequest{
		BookId: "test-book-id",
		UserId: "test-user-id",
	}).Return(&pb.BorrowBookResponse{Message: "Book borrowed successfully"}, nil)

	reqBody := `{"book_id":"test-book-id"}`
	req := httptest.NewRequest(http.MethodPost, "/borrow-book", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+tokenString)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		// Call BorrowBook on the mock client
		res, err := mockClient.BorrowBook(context.TODO(), &pb.BorrowBookRequest{
			BookId: "test-book-id",
			UserId: "test-user-id",
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to borrow book"})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": res.Message})
	}

	// Assertions
	if assert.NoError(t, handler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Book borrowed successfully")
	}
}

// unittest for returning book
func TestReturnBook(t *testing.T) {
	e := echo.New()

	// Mock the JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "test-user-id",
	})
	tokenString, _ := token.SignedString([]byte("12345"))

	// Mock LibraryServiceClient
	mockClient := new(MockLibraryServiceClient)
	mockClient.On("ReturnBook", mock.Anything, &pb.ReturnBookRequest{
		BookId: "test-book-id",
		UserId: "test-user-id",
	}).Return(&pb.ReturnBookResponse{Message: "Book returned successfully"}, nil)

	reqBody := `{"book_id":"test-book-id"}`
	req := httptest.NewRequest(http.MethodPost, "/return-book", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+tokenString)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		// Call ReturnBook on the mock client
		res, err := mockClient.ReturnBook(context.TODO(), &pb.ReturnBookRequest{
			BookId: "test-book-id",
			UserId: "test-user-id",
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to return book"})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": res.Message})
	}

	// Assertions
	if assert.NoError(t, handler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Book returned successfully")
	}
}