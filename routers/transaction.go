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

func newTransactionStruct(associate primitive.ObjectID) *models.Transaction {
	return &models.Transaction{
		AssociateID: associate,
		ID:          primitive.NewObjectID(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func SetupTransactionRoutes(router *gin.Engine) {
	jwtAuthService := utils.GetJWTAuthService()
	transactionRoutes := router.Group("/transactions")
	transactionRoutes.Use(jwtAuthService.AuthMiddleware())
	{
		// Route to get all transaction
		transactionRoutes.GET("/", func(context *gin.Context) {

			filterParam, _ := context.Params.Get("filter")
			pageParam, _ := context.Params.Get("page")
			auth, _ := context.Get("auth")
			authentication := auth.(utils.Authentication)
			pageSize := 10
			page := 1

			if pageParam != "" {
				page, _ = strconv.Atoi(pageParam)
			}

			offset := (page - 1) * pageSize

			// Create a filter for the MongoDB query
			var filter = bson.D{}

			// Add the associate_id filter for non-admin users
			if authentication.Role != "admin" {
				filter = append(filter, bson.E{Key: "associate_id", Value: authentication.ID})
			}

			if filterParam != "" {
				now := time.Now()
				switch filterParam {
				case "today":
					startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
					filter = append(filter, bson.E{Key: "created_at", Value: bson.M{"$gte": startOfDay}})
				case "week":
					startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
					filter = append(filter, bson.E{Key: "created_at", Value: bson.M{"$gte": startOfWeek}})
				case "month":
					startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
					filter = append(filter, bson.E{Key: "created_at", Value: bson.M{"$gte": startOfMonth}})
				case "year":
					startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
					filter = append(filter, bson.E{Key: "created_at", Value: bson.M{"$gte": startOfYear}})
				}
			}

			cursor, err := database.FindDocumentsQuery(models.Collection.Transaction, filter, pageSize, offset)

			if err != nil {
				context.JSON(http.StatusOK, gin.H{
					"message": err.Error(),
				})
				context.Abort()
				return
			}

			var transactions []models.Transaction

			err = cursor.All(context, &transactions)

			if err != nil {
				context.JSON(http.StatusOK, gin.H{
					"message": err.Error(),
				})
				context.Abort()
				return
			}

			context.JSON(http.StatusOK, gin.H{
				"transactions": transactions,
				"page":         page,
			})

			return

		})

		// Route to create new transaction
		transactionRoutes.POST("/", func(context *gin.Context) {
			// TODO: create a new transaction
			auth, _ := context.Get("auth")
			authentication := auth.(utils.Authentication)

			objectId, err := primitive.ObjectIDFromHex(authentication.ID)

			if err != nil {
				context.JSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "message": "You do not have permission to the resource"})
				context.Abort()
				return
			}

			newtransaction := newTransactionStruct(objectId)

			//get body in new transaction
			if err := context.ShouldBindJSON(&newtransaction); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			err = models.ValidateStruct.Struct(newtransaction)

			if err != nil {
				//TODO: return an error response of the required fields left empty
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			insertResult, err := database.InsertDocument(models.Collection.Transaction, utils.ConvertStructPrimitive(newtransaction))

			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				context.Abort()
				return
			}

			context.JSON(http.StatusOK, gin.H{
				"created": insertResult,
				"message": "Successfully added a new transaction",
			})
			return

		})
		// Get info of a transaction
		//not important because all data are returned in the initial get all transaction face
		//transactionRoutes.GET("/:id", func(context *gin.Context) {
		// get a single transaction
		//})
	}
}
