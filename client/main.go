package main

import (
	"context"
	"net/http"

	"os"
	"p3/gc2/config/database"
	book_handler "p3/gc2/handler/bookHandler"
	user_handler "p3/gc2/handler/userHandler"
	cust_middleware "p3/gc2/middleware"
	"p3/gc2/pb"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	// path to swagger docs client and server
	_ "p3/gc2/client/docs"
	"github.com/swaggo/echo-swagger"
)

// BorrowBookRequest represents the request body for borrowing a book
type BorrowBookRequest struct {
    BookID string `json:"book_id" validate:"required"`
}

// ReturnBookRequest represents the request body for returning a book
type ReturnBookRequest struct {
    BookID string `json:"book_id" validate:"required"`
}

// @Summary Borrow a book
// @Description Borrow a book using gRPC
// @Tags Books
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param body body BorrowBookRequest true "Book ID to borrow"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/borrow-book [post]
func BorrowBookHandler(c echo.Context) error {
    // Retrieve the token from the context
    token, ok := c.Get("user").(*jwt.Token)
    if !ok || token == nil {
        return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid or missing token"})
    }

    // Extract claims from the token
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid token"})
    }

    // Extract user_id from claims
    userID, ok := claims["user_id"].(string)
    if !ok {
        return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid user ID"})
    }

    // Bind the incoming book_id from the request body
    var request struct {
        BookID string `json:"book_id" validate:"required"`
    }
    if err := c.Bind(&request); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request format"})
    }

    // Add token to metadata for gRPC request
    md := metadata.Pairs("authorization", "Bearer "+token.Raw)
    ctx := metadata.NewOutgoingContext(context.Background(), md)

    // Connect to the gRPC server
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to connect to gRPC server"})
    }
    defer conn.Close()

    // Create a gRPC client
    client := pb.NewLibraryServiceClient(conn)

    // Call BorrowBook on the gRPC server
    res, err := client.BorrowBook(ctx, &pb.BorrowBookRequest{
        BookId: request.BookID,
        UserId: userID,
    })
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to borrow book", "error": err.Error()})
    }

    // Return the gRPC server's response
    return c.JSON(http.StatusOK, map[string]string{
        "message": res.GetMessage(),
    })
}

// @Summary Return a borrowed book
// @Description Allows a user to return a borrowed book by providing the book ID and JWT token for authentication.
// @Tags Books
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param body body ReturnBookRequest true "Book ID to return"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/return-book [post]
func ReturnBookHandler(c echo.Context) error {
    // Retrieve the token from the context
    token, ok := c.Get("user").(*jwt.Token)
    if !ok || token == nil {
        return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid or missing token"})
    }

    // Extract claims from the token
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid token"})
    }

    // Extract user_id from claims
    userID, ok := claims["user_id"].(string)
    if !ok {
        return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid user ID"})
    }

    // Bind the incoming book_id from the request body
    var request struct {
        BookID string `json:"book_id" validate:"required"`
    }
    if err := c.Bind(&request); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request format"})
    }

    // Add token to metadata for gRPC request
    md := metadata.Pairs("authorization", "Bearer "+token.Raw)
    ctx := metadata.NewOutgoingContext(context.Background(), md)

    // Connect to the gRPC server
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to connect to gRPC server"})
    }
    defer conn.Close()

    // Create a gRPC client
    client := pb.NewLibraryServiceClient(conn)

    // Call ReturnBook on the gRPC server
    res, err := client.ReturnBook(ctx, &pb.ReturnBookRequest{
        BookId: request.BookID,
        UserId: userID,
    })
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to return book", "error": err.Error()})
    }

    // Return the gRPC server's response
    return c.JSON(http.StatusOK, map[string]string{
        "message": res.GetMessage(),
    })
}

// @title Library API
// @version 1.0
// @description API documentation for the library management system.
// @host localhost:8080
// @BasePath /
func main(){
	// populate the db
	// config.MigrateData()

	// get os
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default to 8080
	}

	// connect to db
	config.InitDB()
	defer config.CloseDB()

	// echo controller
	e := echo.New()
	e.Validator = &cust_middleware.CustomValidator{Validator: validator.New()}

	// public routes for users
	e.POST("/users/register", user_handler.RegisterUser)	
	e.POST("/users/login", user_handler.LoginUser)

	// protected routes for users using JWT middleware
	usersGroup := e.Group("/users")
	usersGroup.Use(cust_middleware.JWTMiddleware)

	// routes for admin (which has it's own authentication protection scheme)
	usersGroup.POST("/books/create", book_handler.CreateBook)
	usersGroup.GET("/books/get", book_handler.GetAllBooks)	
	usersGroup.GET("/books/get/:id", book_handler.GetBookByID)
	usersGroup.PUT("/books/:id", book_handler.UpdateBook)
	usersGroup.DELETE("/books/:id", book_handler.DeleteBook)

	// gRPC route
	usersGroup.POST("/borrow-book", BorrowBookHandler)
	usersGroup.POST("/return-book", ReturnBookHandler)
	
	// Add this route for Swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	
	e.Logger.Fatal(e.Start(":" + port))
}