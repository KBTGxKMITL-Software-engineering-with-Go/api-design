package main

import (
	"fmt"
	echojwt "github.com/labstack/echo-jwt/v4"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Err struct {
	Message string `json:"message"`
}

var users = []User{
	{ID: 1, Name: "AnuchitO", Age: 18},
}

func createUserHandler(c echo.Context) error {
	u := User{}
	err := c.Bind(&u)
	if err != nil {
		return c.JSON(http.StatusBadRequest, Err{Message: err.Error()})
	}

	users = append(users, u)

	fmt.Println("id : % #v\n", u)

	return c.JSON(http.StatusCreated, u)
}

func getUsersHandler(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*jwtCustomClaims)
	//name := claims.Name
	fmt.Printf("User Name : %#v\n", claims)

	return c.JSON(http.StatusOK, users)
}

type jwtCustomClaims struct {
	Name string `json:"name"`
	//Admin bool   `json:"admin"`
	Role string `json:"role"`
	Type string `json:"type"` // Added to distinguish between access and refresh tokens
	jwt.RegisteredClaims
}

func login(c echo.Context) error {

	username := c.FormValue("username")
	password := c.FormValue("password")

	// Throws unauthorized error
	if username != "jon" || password != "shhh!" {
		return echo.ErrUnauthorized
	}

	// Set custom claims
	claims := &jwtCustomClaims{
		"Jon Snow",
		"admin",
		"access",
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 5)),
		},
	}

	// Create token with claims
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	accessTokenString, err := accessToken.SignedString([]byte("secret"))
	if err != nil {
		return err
	}

	// Create refresh token
	refreshTokenClaims := &jwtCustomClaims{
		"Jon Snow",
		"admin",
		"refresh",
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Longer-lived
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte("secret"))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"access_token":  accessTokenString,
		"refresh_token": refreshTokenString,
	})
}

// refreshToken function
func refreshToken(c echo.Context) error {
	refreshTokenString := c.FormValue("refresh_token")

	jwtSecretKey := []byte("secret")

	// Parse the token
	token, err := jwt.ParseWithClaims(refreshTokenString, &jwtCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil
	})

	if err != nil {
		return echo.ErrUnauthorized
	}

	claims, ok := token.Claims.(*jwtCustomClaims)
	if !ok || !token.Valid || claims.Type != "refresh" {
		return echo.ErrUnauthorized
	}

	// Create new access token
	newAccessTokenClaims := &jwtCustomClaims{
		claims.Name,
		claims.Role,
		"access",
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		},
	}
	newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newAccessTokenClaims)
	newAccessTokenString, err := newAccessToken.SignedString(jwtSecretKey)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"access_token":  newAccessTokenString,
		"refresh_token": refreshTokenString,
	})
}
func roleCheckMiddleware(requiredRole string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get("user").(*jwt.Token)
			claims := user.Claims.(*jwtCustomClaims)
			//!= requiredRole
			fmt.Printf("User Name : %#v\n", claims)
			fmt.Println(claims.Role)
			if claims.Role != requiredRole {
				return echo.ErrUnauthorized
			}

			return next(c)
		}
	}
}
func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	g := e.Group("/api")
	g.POST("/login", login)
	e.POST("/refresh", refreshToken)

	u := g.Group("/users")
	config := echojwt.Config{
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(jwtCustomClaims)
		},
		SigningKey: []byte("secret"),
	}
	u.Use(echojwt.WithConfig(config))
	u.Use(adminMiddleware("admin"))
	u.POST("", createUserHandler)
	u.GET("", getUsersHandler)

	log.Fatal(e.Start(":2565"))
}

func adminMiddleware(requiredRole string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get("user").(*jwt.Token)
			claims := user.Claims.(*jwtCustomClaims)
			fmt.Printf("\n\n\nUser Name : %#v\n\n\n", claims)
			if claims.Role != requiredRole {
				return echo.ErrUnauthorized
			}

			return next(c)
		}
	}
}
