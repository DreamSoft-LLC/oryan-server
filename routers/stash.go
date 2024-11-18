package routers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DreamSoft-LLC/oryan/database"
	"github.com/DreamSoft-LLC/oryan/models"
	"github.com/DreamSoft-LLC/oryan/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func newStashStruct(associate primitive.ObjectID) *models.Stash {
	return &models.Stash{
		AssociateID: associate,
		ID:          primitive.NewObjectID(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func SetupStashRoutes(router *gin.Engine) {
	authService := utils.GetJWTAuthService()

	stashRoutes := router.Group("/stash")
	stashRoutes.Use(authService.AuthMiddleware())
	{

		stashRoutes.GET("", func(c *gin.Context) {

			pageParam := c.Query("page")
			pageSize := 1000000000
			page := 1

			if pageParam != "" {
				page, _ = strconv.Atoi(pageParam)
			}

			var filter = bson.D{}

			offset := (page - 1) * pageSize

			cursor, err := database.FindDocumentsQuery(models.Collection.Stash, filter, pageSize, offset)

			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"message": err.Error(),
				})
				c.Abort()
				return
			}

			var stashs []models.Stash

			err = cursor.All(c, &stashs)

			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"message": err.Error(),
				})
				c.Abort()
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"stashs": stashs,
			})

		})

		stashRoutes.POST("", func(c *gin.Context) {

			auth, _ := c.Get("auth")
			authentication := auth.(*utils.Authentication)

			idStr := authentication.ID
			if strings.HasPrefix(idStr, "ObjectID(") && strings.HasSuffix(idStr, ")") {
				idStr = idStr[9 : len(idStr)-1]
			}
			idStr = strings.Trim(idStr, "\"")

			objectId, err := primitive.ObjectIDFromHex(idStr)

			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Invalid authentication ID",
					"message": "You do not have permission to access the resource",
				})
				c.Abort()
				return
			}
			newShash := newStashStruct(objectId)

			if err := c.ShouldBindJSON(&newShash); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				c.Abort()
				return
			}

			err = models.ValidateStruct.Struct(newShash)

			if err != nil {
				//TODO: return an error response of the required fields left empty
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				c.Abort()
				return
			}

			insertResult, err := database.InsertDocument(models.Collection.Stash, utils.ConvertStructPrimitive(newShash))

			if err != nil {
				//TODO: return an error response of the required fields left empty
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				c.Abort()
				return
			}

			newShash.ID = insertResult.InsertedID.(primitive.ObjectID)

			c.JSON(http.StatusOK, gin.H{
				"stash":   newShash,
				"message": "You have successfully created a new stash",
			})

			return

		})
	}

}
