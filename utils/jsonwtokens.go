package utils

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type Configuration struct {
	JwtSecret     string
	JwtExpiration string
}

type JWTAuthService struct {
	Config *Configuration
}

func NewJWTAuthService(cfg *Configuration) *JWTAuthService {
	return &JWTAuthService{
		Config: cfg,
	}
}

// AuthMiddleware is the JWT authentication middleware for Gin
func (j *JWTAuthService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized - no token given"})
			c.Abort()
			return
		}
		parts := strings.Split(tokenString, " ")
		token := parts[len(parts)-1]

		valid, err := j.ValidateJWT(token)
		if err != nil {
			println(err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized - invalid token"})
			c.Abort()
			return
		}

		if !valid {
			log.Printf("Invalid token: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized - invalid token"})
			c.Abort()
			return
		}

		claims, err := j.DecodeJWT(token)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized - invalid token"})
			c.Abort()
			return
		}
		// Store claims in the context
		c.Set("auth", claims)
		c.Next()
	}
}

type AuthenticationClaims struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type Authentication struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// DecodeJWT extracts token data (claims)
func (j *JWTAuthService) DecodeJWT(token string) (*Authentication, error) {
	claim := &AuthenticationClaims{}
	tokenData, err := jwt.ParseWithClaims(token, claim, func(t *jwt.Token) (interface{}, error) {
		return []byte(j.Config.JwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// Check if the token is valid and claims were correctly parsed
	if tokenData.Valid {
		auth := &Authentication{
			ID:    claim.ID,
			Email: claim.Email,
			Role:  claim.Role,
		}

		return auth, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateJWT validates an existing token
func (j *JWTAuthService) ValidateJWT(token string) (bool, error) {

	claim := &AuthenticationClaims{}

	tokenData, err := jwt.ParseWithClaims(token, claim, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.Config.JwtSecret), nil
	})

	if err != nil {
		return false, err
	}

	if _, ok := tokenData.Claims.(*AuthenticationClaims); ok && tokenData.Valid {
		return true, nil
	}
	return false, nil
}

func (j *JWTAuthService) SignJWT(claims AuthenticationClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.Config.JwtSecret))
}

var (
	jwtAuthService *JWTAuthService
	once           sync.Once
)

func GetJWTAuthService() *JWTAuthService {
	once.Do(func() {
		config := &Configuration{
			JwtSecret:     os.Getenv("JWT_SECRET"),
			JwtExpiration: "12h",
		}
		jwtAuthService = NewJWTAuthService(config)
	})
	return jwtAuthService
}
