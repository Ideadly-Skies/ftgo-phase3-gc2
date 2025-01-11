package handler

import (
	"context"
	"fmt"
	"net/http"
	config "p3/gc2/config/database"
	cust_middleware "p3/gc2/middleware"

	"time"

	"github.com/labstack/echo/v4"
	"github.com/google/uuid"
)

// Book struct to temporarily store book information
type Book struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Author        string    `json:"author"`
	PublishedDate time.Time `json:"published_date"`
	Status        string    `json:"status"`
	UserID        *string    `json:"user_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Request struct for creating/updating a book
type BookRequest struct {
	Title         string    `json:"title" validate:"required"`
	Author        string    `json:"author" validate:"required"`
	PublishedDate string 	`json:"published_date" validate:"required"`
}

// Response struct for success messages
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// CreateBook handler
// @Summary Create a new book
// @Description Create a new book with title, author, and published date
// @Tags Books
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param body body BookRequest true "Book data"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /books/create [post]
func CreateBook(c echo.Context) error {
	if !cust_middleware.IsAdmin(c) {
		return c.JSON(http.StatusForbidden, map[string]string{"message": "Permission denied admin use only!"})
	}

	var req BookRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request"})
	}

	// Validate the request body
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Validation failed", "error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Generate a new UUID for the book
	bookID := uuid.New().String()

	// Query to insert the book into the database
	query := `INSERT INTO books (id, title, author, published_date, status) VALUES ($1, $2, $3, $4, 'Available')`
	_, err := config.Pool.Exec(ctx, query, bookID, req.Title, req.Author, req.PublishedDate)
	if err != nil {
		fmt.Println("Error inserting into books table:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to create book"})
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Message: "Book created successfully",
		Data:    map[string]string{"id": bookID},
	})
}

// GetAllBooks handler
// @Summary Get all books
// @Description Retrieve all books with their details
// @Tags Books
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} map[string]string
// @Router /books/get [get]
func GetAllBooks(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query to get all books from the database
	query := `SELECT id, title, author, published_date, status, user_id, created_at, updated_at FROM books`
	rows, err := config.Pool.Query(ctx, query)
	if err != nil {
		fmt.Println("Error fetching books:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to fetch books"})
	}
	defer rows.Close()

	// Iterate through the rows and build the response
	var books []Book
	for rows.Next() {
		var book Book
		if err := rows.Scan(&book.ID, &book.Title, &book.Author, &book.PublishedDate, &book.Status, &book.UserID, &book.CreatedAt, &book.UpdatedAt); err != nil {
			fmt.Println("Error scanning book:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to parse books"})
		}
		books = append(books, book)
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Message: "Books fetched successfully",
		Data:    books,
	})
}

// GetBookByID handler
// @Summary Get book by ID
// @Description Retrieve details of a specific book by its ID
// @Tags Books
// @Produce json
// @Param id path string true "Book ID"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /books/get/{id} [get]
func GetBookByID(c echo.Context) error {
	bookID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query to get a specific book by ID
	query := `SELECT id, title, author, published_date, status, user_id, created_at, updated_at FROM books WHERE id = $1`
	var book Book
	err := config.Pool.QueryRow(ctx, query, bookID).Scan(&book.ID, &book.Title, &book.Author, &book.PublishedDate, &book.Status, &book.UserID, &book.CreatedAt, &book.UpdatedAt)
	if err != nil {
		fmt.Println("Error fetching book:", err)
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Book not found"})
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Message: "Book fetched successfully",
		Data:    book,
	})
}

// UpdateBook handler
// @Summary Update book details
// @Description Update the details of a specific book
// @Tags Books
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Book ID"
// @Param body body BookRequest true "Book data"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /books/{id} [put]
func UpdateBook(c echo.Context) error {
	if !cust_middleware.IsAdmin(c) {
		return c.JSON(http.StatusForbidden, map[string]string{"message": "Permission denied admin use only!"})
	}
	
	bookID := c.Param("id")
	var req BookRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request"})
	}

	// Validate the request body
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Validation failed", "error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query to update the book details
	query := `UPDATE books SET title = $1, author = $2, published_date = $3, updated_at = NOW() WHERE id = $4`
	_, err := config.Pool.Exec(ctx, query, req.Title, req.Author, req.PublishedDate, bookID)
	if err != nil {
		fmt.Println("Error updating book:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to update book"})
	}

	return c.JSON(http.StatusOK, SuccessResponse{Message: "Book updated successfully"})
}

// DeleteBook handler
// @Summary Delete a book
// @Description Delete a book by its ID
// @Tags Books
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Book ID"
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /books/{id} [delete]
func DeleteBook(c echo.Context) error {
	if !cust_middleware.IsAdmin(c) {
		return c.JSON(http.StatusForbidden, map[string]string{"message": "Permission denied admin use only!"})
	}

	bookID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query to delete the book by ID
	query := `DELETE FROM books WHERE id = $1`
	_, err := config.Pool.Exec(ctx, query, bookID)
	if err != nil {
		fmt.Println("Error deleting book:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to delete book"})
	}

	return c.JSON(http.StatusOK, SuccessResponse{Message: "Book deleted successfully"})
}