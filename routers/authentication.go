package routers

import (
	"errors"
	"github.com/DreamSoft-LLC/oryan/database"
	"github.com/DreamSoft-LLC/oryan/models"
	"github.com/DreamSoft-LLC/oryan/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
)

type requestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func SetupAuthRoutes(router *gin.Engine) {

	authRoutes := router.Group("/auth")
	{

		authRoutes.POST("/login", func(c *gin.Context) {
			var body requestBody
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				c.Abort()
				return
			}

			//check if fields are not empty
			if body.Email == "" || body.Password == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "email or password is required"})
				c.Abort()
				return
			}

			var result models.Associate

			//	check if credential in Associate
			err := database.FindDocument(models.Collection.Associate, bson.D{{"email", body.Email}}).Decode(&result)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid login credentials"})

				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Internal server error"})
				}
				c.Abort()
				return
			}

			//	check if password match
			if !utils.CheckPasswordHash(body.Password, result.Password) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid login credentials"})
				c.Abort()
				return
			}

			authClaim := utils.AuthenticationClaims{
				ID:    result.ID.String(),
				Role:  result.Role,
				Email: result.Email,
			}
			authService := utils.GetJWTAuthService()
			secureToken, err := authService.SignJWT(authClaim)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				c.Abort()
				return
			}

			result.Password = ""

			c.JSON(http.StatusOK, gin.H{
				"auth_token": secureToken,
				"user_info":  result,
				"message":    "Login Success",
			})
			return
		})

	}
}
