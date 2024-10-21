package routers

import (
	"fmt"
	"net/http"
	"os"
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

func newMiscellaneousStruct(associate primitive.ObjectID) *models.Miscellaneous {
	return &models.Miscellaneous{
		ID:          primitive.NewObjectID(),
		AssociateID: associate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func SetupMiscellaneousRoutes(router *gin.Engine) {
	jwtAuthService := utils.GetJWTAuthService()
	miscellaneousRoutes := router.Group("/miscellaneous")
	miscellaneousRoutes.Use(jwtAuthService.AuthMiddleware())
	{

		miscellaneousRoutes.GET("/", func(context *gin.Context) {

			filterParam := context.Query("filter")
			pageParam := context.Query("page")
			// auth, _ := context.Get("auth")
			// authentication := auth.(*utils.Authentication)
			pageSize := 50
			page := 1

			fmt.Printf("Filter: %+v\n", filterParam)

			if pageParam != "" {
				page, _ = strconv.Atoi(pageParam)
			}

			offset := (page - 1) * pageSize

			// Create a filter for the MongoDB query
			var filter = bson.D{}

			// Add the associate_id filter for non-admin users
			// if authentication.Role != "admin" {
			// 	filter = append(filter, bson.E{Key: "associate_id", Value: authentication.ID})
			// }

			now := time.Now()
			if filterParam != "" {
				switch filterParam {
				case "today":
					startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
					filter = append(filter, bson.E{Key: "created_date", Value: bson.M{"$gte": startOfDay}})
				case "week":
					startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
					filter = append(filter, bson.E{Key: "created_date", Value: bson.M{"$gte": startOfWeek}})
				case "month":
					startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
					filter = append(filter, bson.E{Key: "created_date", Value: bson.M{"$gte": startOfMonth}})
				case "year":
					startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
					filter = append(filter, bson.E{Key: "created_date", Value: bson.M{"$gte": startOfYear}})
				}
			}

			cursor, err := database.FindDocumentsQuery(models.Collection.Miscellaneous, filter, pageSize, offset)

			if err != nil {
				context.JSON(http.StatusOK, gin.H{
					"message": err.Error(),
				})
				context.Abort()
				return
			}

			var miscellaneousTransactions []models.Miscellaneous

			err = cursor.All(context, &miscellaneousTransactions)

			if err != nil {
				context.JSON(http.StatusOK, gin.H{
					"message": err.Error(),
				})
				context.Abort()
				return
			}

			context.JSON(http.StatusOK, gin.H{
				"miscellaneous": miscellaneousTransactions,
			})

		})

		miscellaneousRoutes.POST("/", func(context *gin.Context) {

			balanceDocumentID := os.Getenv("BALANCE_ID")
			if balanceDocumentID == "" {
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Balance document ID not set"})
				return
			}

			auth, _ := context.Get("auth")
			authentication := auth.(*utils.Authentication)

			idStr := authentication.ID
			if strings.HasPrefix(idStr, "ObjectID(") && strings.HasSuffix(idStr, ")") {
				idStr = idStr[9 : len(idStr)-1]
			}
			idStr = strings.Trim(idStr, "\"")

			objectId, err := primitive.ObjectIDFromHex(idStr)

			if err != nil {
				context.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Invalid authentication ID",
					"message": "You do not have permission to access the resource",
				})
				context.Abort()
				return
			}

			// Initialize the struct
			body := newMiscellaneousStruct(objectId)

			// Bind incoming JSON to struct
			if err := context.ShouldBindJSON(&body); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Validate the struct
			if err := models.ValidateStruct.Struct(body); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Insert document into Miscellaneous collection
			insertResult, err := database.InsertDocument(models.Collection.Miscellaneous, utils.ConvertStructPrimitive(body))
			if err != nil {
				context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				context.Abort()
				return
			}

			// Retrieve the balance document by ID (ensure `balance_docucment_id` is initialized correctly)
			var balanceDoc models.Balance
			docResult := database.FindDocumentById(models.Collection.Balance, balanceDocumentID)
			if err := docResult.Decode(&balanceDoc); err != nil {
				context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				context.Abort()
				return
			}

			// Parse the amounts as floats for calculation
			newAmount, err := strconv.ParseFloat(body.Amount, 64)
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount format"})
				return
			}

			balanceAmountSpent, err := strconv.ParseFloat(balanceDoc.Spent, 64)
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid balance amount format"})
				return
			}

			// Update the balance document by subtracting the new amount
			s := fmt.Sprintf("%f", balanceAmountSpent+newAmount)

			updateResult, err := database.UpdateDocument(
				models.Collection.Balance,
				bson.D{{Key: "_id", Value: balanceDocumentID}},
				bson.D{{Key: "$set", Value: bson.D{{Key: "amount", Value: s}}}},
			)
			if err != nil {
				context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				context.Abort()
				return
			}

			// Assign the inserted document ID to the body struct
			body.ID = insertResult.InsertedID.(primitive.ObjectID)

			// Return the response with the updated balance
			context.JSON(http.StatusOK, gin.H{
				"miscellaneous": body,
				"updateResult":  updateResult,
			})

		})

	}
}
