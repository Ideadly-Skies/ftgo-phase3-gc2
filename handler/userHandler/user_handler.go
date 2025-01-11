package handler

import (
	"fmt"
	"net/http"
	config "p3/gc2/config/database"

	"context"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"github.com/jackc/pgconn"
	"github.com/google/uuid"
)	

//  user struct to handle temporarily store customer information
type User struct {
	ID   	  string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Role	  string `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdateAt  time.Time `json:"update_at"`
	Jwt 	  string `json:"jwt"`
}

// RegisterRequest for user
type RegisterRequest struct {
	Username string `json:"username" validate:"required,username"`
	Password string `json:"password" validate:"required,password"`
	Role 	 string `josn:"role" validate:"required,role"`
}

// LoginRequest for user
type LoginRequest struct {
	Username string `json:"username" validate:"required,username"`
	Password string `json:"password" validate:"required,password"`
}

// login response: token
type LoginResponse struct {
	Token string `json:"token"`
}

var jwtSecret = []byte("12345")

/* user route */

// @Summary Register a new user
// @Description Register a new user with username, password, and role
// @Tags Users
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User registration request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/register [post]
func RegisterUser(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": err.Error()})
	}

	// hash the password
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal Server Error"})
	}
	
	// query to insert into users db
	user_query := "INSERT INTO users (id, username, password, role) VALUES ($1, $2, $3, $4)"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// query row: inserting new user to users table 
	id := uuid.New() // generate new uuid for the user
	_, err = config.Pool.Exec(ctx, user_query, id.String(), req.Username, string(hashPassword), req.Role)
	if err != nil {
		fmt.Println("Error inserting into users table: ", err)

		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" {
				return c.JSON(http.StatusBadRequest, map[string]string{"message": "Username already registered"})
			}
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal Server Error"})
	}

	// return successful user registration message
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf(`User %s registered successfully`,req.Username),
        "username": req.Username,
	})
}

// @Summary Login user
// @Description Authenticates a user and returns a JWT token for subsequent requests.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/login [post]
func LoginUser(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message":"Invalid Request"})
	}

	var user User 
	query := "SELECT id, username, password, role FROM users WHERE username = $1"
	
	err := config.Pool.QueryRow(context.Background(), query, req.Username).Scan(&user.ID, &user.Username, &user.Password, &user.Role)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid username inputted"})
	}

	// compare password to see if it matches the user password provided
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid password inputted"})
	}

	// create new jwt claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"password": user.Password,
		"role": user.Role,
		"exp": jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
	})
	
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Invalid Generate Token"})
	}

	// Update the jwt_token column in the database
	updateQuery := "UPDATE users SET jwt_token = $1 WHERE id = $2"
	_, err = config.Pool.Exec(context.Background(), updateQuery, tokenString, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	// return ok status and login response
	return c.JSON(http.StatusOK, LoginResponse{Token: tokenString})
}