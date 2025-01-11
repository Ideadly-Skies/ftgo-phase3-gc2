package main

import (
	"context"
	// "fmt"
	// "fmt"
	// "log"
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
)

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

	// borrow book (gRPC process)
	usersGroup.POST("/borrow-book", func(c echo.Context) error {
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
	})
	
	// return book (gRPC process)
	usersGroup.POST("/return-book", func(c echo.Context) error {
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
	})

	e.Logger.Fatal(e.Start(":" + port))
}