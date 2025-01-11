package main

import (
	"context"
	"log"
	"net"
	"time"
	
	"p3/gc2/config/database"
	"p3/gc2/pb"
	"os"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"github.com/robfig/cron/v3"
)

// LibraryServer implements the gRPC service defined in the proto file.
type LibraryServer struct {
	pb.UnimplementedLibraryServiceServer
}

// Job to update overdue books
func updateOverdueBooks() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update all books that are overdue
	query := `
		UPDATE books
		SET status = 'Missing'
		WHERE id IN (
			SELECT book_id
			FROM borrowedbooks
			WHERE return_date IS NULL
			  AND borrowed_date < NOW() - INTERVAL '3 weeks'
		)`
	res, err := config.Pool.Exec(ctx, query)
	if err != nil {
		log.Printf("Error updating overdue books: %v\n", err)
		return
	}

	rowsAffected := res.RowsAffected()
	log.Printf("Job completed: Updated %d overdue books to 'Missing'\n", rowsAffected)
}

// BorrowBook handles the gRPC request to borrow a book.
func (s *LibraryServer) BorrowBook(ctx context.Context, req *pb.BorrowBookRequest) (*pb.BorrowBookResponse, error) {
	// Check if metadata contains the authorization token
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md["authorization"]) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
	}

	// Validate the token
	tokenStr := md["authorization"][0]
	// Remove "Bearer " prefix
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}
	jwtSecret := []byte("12345")

	_, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		log.Printf("Invalid token: %v", err)
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	bookID := req.GetBookId()
	userID := req.GetUserId()

	// Check if the book is available
	var bookStatus string
	err = config.Pool.QueryRow(context.Background(), `SELECT status FROM books WHERE id = $1`, bookID).Scan(&bookStatus)
	if err != nil {
		return nil, status.Error(codes.NotFound, "book not found")
	}

	if bookStatus != "Available" {
		return nil, status.Error(codes.FailedPrecondition, "book is not available")
	}

	// Borrow the book: Update the status and create an entry in BorrowedBooks
	tx, err := config.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE books SET status = 'Borrowed', user_id = $1 WHERE id = $2`, userID, bookID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update book status")
	}

	_, err = tx.Exec(ctx, `INSERT INTO borrowedbooks (book_id, user_id, borrowed_date) VALUES ($1, $2, NOW())`, bookID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to log borrowed book")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "failed to commit transaction")
	}

	return &pb.BorrowBookResponse{
		Message: "Book borrowed successfully",
	}, nil
}

func (s *LibraryServer) ReturnBook(ctx context.Context, req *pb.ReturnBookRequest) (*pb.ReturnBookResponse, error) {
	// Check if metadata contains the authorization token
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md["authorization"]) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
	}

	// Validate the token
	tokenStr := md["authorization"][0]
	// Remove "Bearer " prefix
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}
	jwtSecret := []byte("12345")

	_, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		log.Printf("Invalid token: %v", err)
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	bookID := req.GetBookId()
	userID := req.GetUserId()

	// Check if the book is currently borrowed by the user
	var dbUserID string
	err = config.Pool.QueryRow(context.Background(), `SELECT user_id FROM books WHERE id = $1 AND status = 'Borrowed'`, bookID).Scan(&dbUserID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "book not found or not borrowed")
	}

	if dbUserID != userID {
		return nil, status.Error(codes.PermissionDenied, "book not borrowed by this user")
	}

	// Return the book: Update the status and borrowed date
	tx, err := config.Pool.Begin(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE books SET status = 'Available', user_id = NULL WHERE id = $1`, bookID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update book status")
	}

	_, err = tx.Exec(ctx, `UPDATE borrowedbooks SET return_date = NOW() WHERE book_id = $1 AND user_id = $2`, bookID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to log return book")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "failed to commit transaction")
	}

	return &pb.ReturnBookResponse{
		Message: "Book returned successfully",
	}, nil
}

// UnaryAuthInterceptor is a gRPC interceptor for token validation.
func UnaryAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Perform token validation for every request
	ctx, err := AuthInterceptor(ctx)
	if err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

// AuthInterceptor validates the JWT token in the metadata.
func AuthInterceptor(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Println("No metadata found")
		return nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
	}

	token := md["authorization"]
	if len(token) == 0 {
		log.Println("Invalid or Missing token")
		return nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
	}

	log.Println("Token validated successfully")
	return ctx, nil
}

func main() {
	// Initialize database connection
	config.InitDB()
	defer config.CloseDB()

	// Start the job scheduler
	c := cron.New()
	_, err := c.AddFunc("@daily", updateOverdueBooks) // Schedule the job to run daily
	if err != nil {
		log.Fatalf("Failed to schedule cron job: %v", err)
	}
	c.Start()
	defer c.Stop()

	// Get the port from the environment variable (default to 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryAuthInterceptor),
	)

	// Register LibraryService
	pb.RegisterLibraryServiceServer(grpcServer, &LibraryServer{})

	log.Println("Server is running on port 50051...")
	if err := grpcServer.Serve(listen); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}