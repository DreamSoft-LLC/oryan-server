package routers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/DreamSoft-LLC/oryan/database"
	"github.com/DreamSoft-LLC/oryan/models"
	"github.com/DreamSoft-LLC/oryan/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

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
		transactionRoutes.GET("", func(context *gin.Context) {

			filterParam := context.Query("filter")
			pageParam := context.Query("page")
			auth, _ := context.Get("auth")
			authentication := auth.(*utils.Authentication)
			pageSize := 10
			page := 1

			fmt.Printf("Filter: %+v\n", filterParam)

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
			now := time.Now()
			if filterParam != "" {
				switch filterParam {
				case "today":
					startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
					filter = append(filter, bson.E{Key: "created_at", Value: bson.M{"$gte": startOfDay}})
				case "week":
					startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
					filter = append(filter, bson.E{Key: "created_at", Value: bson.M{"$gte": startOfWeek}})
				case "month":
					startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
					filter = append(filter, bson.E{Key: "created_at", Value: bson.M{"$gte": startOfMonth}})
				case "year":
					startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
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
			authentication := auth.(*utils.Authentication)

			balanceDocumentID := os.Getenv("BALANCE_ID")

			if balanceDocumentID == "" {
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Balance document ID not set"})
				return
			}

			idStr := authentication.ID
			if strings.HasPrefix(idStr, "ObjectID(") && strings.HasSuffix(idStr, ")") {
				idStr = idStr[9 : len(idStr)-1]
			}
			idStr = strings.Trim(idStr, "\"")

			objectId, err := primitive.ObjectIDFromHex(idStr)

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

			newtransaction.ID = insertResult.InsertedID.(primitive.ObjectID)

			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				context.Abort()
				return
			}

			fixedID, _ := primitive.ObjectIDFromHex(balanceDocumentID)
			var balanceData models.Balance
			balanceDocument := database.FindDocument(models.Collection.Balance, bson.D{{Key: "_id", Value: fixedID}})

			err = balanceDocument.Decode(&balanceData)

			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				context.Abort()
				return
			}

			balanceAmountSpent, _ := strconv.ParseFloat(balanceData.Spent, 64)
			transactionAmount, _ := strconv.ParseFloat(newtransaction.Amount, 64)

			if newtransaction.Kind == "buy" {
				s := fmt.Sprintf("%f", balanceAmountSpent+transactionAmount)

				_, err := database.UpdateDocument(models.Collection.Balance, bson.D{{Key: "_id", Value: fixedID}}, bson.D{{Key: "$set", Value: bson.D{{Key: "spent", Value: s}}}})

				if err != nil {
					context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					context.Abort()
					return
				}

			}

			context.JSON(http.StatusOK, gin.H{
				"created":     insertResult,
				"transaction": newtransaction,
				"message":     "Successfully added a new transaction",
			})
			return

		})

		transactionRoutes.GET("/scales", func(context *gin.Context) {

			auth, _ := context.Get("auth")
			filterParam := context.Query("filter")
			authentication := auth.(*utils.Authentication)

			// Ensure the user is an admin
			if authentication.Role != "admin" {
				context.JSON(http.StatusUnauthorized, gin.H{
					"message": "you do not have admin rights",
				})
				context.Abort()
				return
			}

			now := time.Now()

			// Build the filter using bson.M
			filter := bson.M{}

			if filterParam != "" {
				switch filterParam {
				case "today":
					startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
					filter["created_at"] = bson.M{"$gte": startOfDay}
				case "week":
					startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
					filter["created_at"] = bson.M{"$gte": startOfWeek}
				case "month":
					startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
					filter["created_at"] = bson.M{"$gte": startOfMonth}
				case "year":
					startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
					filter["created_at"] = bson.M{"$gte": startOfYear}
				}
			}

			// Initialize a map to hold the results for each scale (using lowercase)
			results := make(map[string]primitive.Decimal128)

			// Sum all scale transactions with the filter
			err := database.SumAllScaleTransactions(models.Collection.Transaction, filter, "amount", results)
			if err != nil {
				context.JSON(http.StatusInternalServerError, gin.H{
					"message": "Error fetching scale transactions",
					"error":   err.Error(),
				})
				return
			}

			// Return the results as JSON (lowercase keys)
			context.JSON(http.StatusOK, gin.H{
				"bb":   results["bb"],   // lowercase for bb
				"mini": results["mini"], // lowercase for mini
				"gb":   results["gb"],   // lowercase for gb
			})
		})

	}
}
