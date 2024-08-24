package routers

import (
	"github.com/DreamSoft-LLC/oryan/database"
	"github.com/DreamSoft-LLC/oryan/models"
	"github.com/DreamSoft-LLC/oryan/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strconv"
	"time"
)

func newCustomerStruct(associate primitive.ObjectID) *models.Customer {
	return &models.Customer{
		CreatedBy: associate,
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func SetupClientRoutes(router *gin.Engine) {
	authService := utils.GetJWTAuthService()

	router.Group("/client")
	router.Use(authService.AuthMiddleware())

	router.GET("/", func(c *gin.Context) {

		pageParam, _ := c.Params.Get("page")
		pageSize := 10
		page := 1

		if pageParam != "" {
			page, _ = strconv.Atoi(pageParam)
		}

		offset := (page - 1) * pageSize

		cursor, err := database.FindDocumentsQuery(models.Collection.Transaction, bson.D{}, pageSize, offset)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		var customers []models.Customer

		err = cursor.All(c, &customers)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"customers": customers,
		})

		return

	})

	router.POST("/", func(c *gin.Context) {

		auth, _ := c.Get("auth")
		authentication := auth.(utils.Authentication)

		objectId, err := primitive.ObjectIDFromHex(authentication.ID)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "message": "You do not have permission to the resource"})
			c.Abort()
			return
		}
		newCustomer := newCustomerStruct(objectId)

		if err := c.ShouldBindJSON(&newCustomer); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		err = models.ValidateStruct.Struct(newCustomer)

		if err != nil {
			//TODO: return an error response of the required fields left empty
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		insertResult, err := database.InsertDocument(models.Collection.Customer, utils.ConvertStructPrimitive(newCustomer))

		if err != nil {
			//TODO: return an error response of the required fields left empty
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		newCustomer.ID = insertResult.InsertedID.(primitive.ObjectID)

		c.JSON(http.StatusOK, gin.H{
			"customer": newCustomer,
			"message":  "You have successfully created a new customer",
		})

		return

	})

	router.GET("/:id", func(c *gin.Context) {

	})

}
