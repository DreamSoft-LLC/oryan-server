package routers

import (
	"github.com/DreamSoft-LLC/oryan/database"
	"github.com/DreamSoft-LLC/oryan/middlewares"
	"github.com/DreamSoft-LLC/oryan/models"
	"github.com/DreamSoft-LLC/oryan/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"
)

func newAssociateStruct() *models.Associate {
	return &models.Associate{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func SetupAssociatesRoutes(router *gin.Engine) {

	authService := utils.GetJWTAuthService()
	associateRoutes := router.Group("/associates")
	//TODO: validate is user has permission to add an associate
	associateRoutes.Use(authService.AuthMiddleware())

	//TODO: validate user level access
	{
		// create a new associate route
		associateRoutes.POST("/", middlewares.IsAdminValidate(), func(c *gin.Context) {

			body := newAssociateStruct()
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// ValidateStruct that required fields are present
			err := models.ValidateStruct.Struct(body)

			if err != nil {
				//TODO: return an error response of the required fields left empty
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			securePassword, err := utils.HashPassword(body.Password)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			body.Password = securePassword

			//create user here
			insertResult, err := database.InsertDocument(models.Collection.Associate, utils.ConvertStructPrimitive(body))
			if err != nil {
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"created": insertResult,
				"message": "Successfully added a new associate",
			})

			return
		})

		//get list of associate route
		associateRoutes.GET("/", middlewares.IsAdminValidate(), func(c *gin.Context) {
			//TODO: validate is user has permission to get associate list

			//get all associate
			var associates []models.Associate

			dataCursor, err := database.FindDocuments(models.Collection.Associate, bson.D{})

			if err != nil {
				return
			}

			if err = dataCursor.All(c, &associates); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode associates"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"associates": associates,
			})
			return
		})

		// get an associate record
		associateRoutes.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			var associate models.Associate

			docResult := database.FindDocumentById(models.Collection.Associate, id)

			err := docResult.Decode(&associate)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				c.Abort()
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"associate": associate,
			})
		})
		return

	}
}
