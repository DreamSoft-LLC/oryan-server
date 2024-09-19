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
	"go.mongodb.org/mongo-driver/mongo"
)

func newBalanceStruct() *models.Balance {
	return &models.Balance{
		ID:        primitive.NewObjectID(),
		Spent:     "0",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newFundStruct(associate primitive.ObjectID) *models.Fund {
	return &models.Fund{
		ID:          primitive.NewObjectID(),
		AssociateID: associate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func SetupBalancesRoutes(router *gin.Engine) {
	jwtAuthService := utils.GetJWTAuthService()
	balanceRoutes := router.Group("/balance")
	balanceRoutes.Use(jwtAuthService.AuthMiddleware())
	{

		balanceRoutes.GET("/", func(context *gin.Context) {

			balanceDocumentID := os.Getenv("BALANCE_ID")

			if balanceDocumentID == "" {
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Balance document ID not set"})
				return
			}

			fixedID, err := primitive.ObjectIDFromHex(balanceDocumentID)

			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid balance document ID"})
				return
			}

			var balanceDoc models.Balance

			docResult := database.FindDocument(models.Collection.Balance, bson.D{{Key: "_id", Value: fixedID}})

			err = docResult.Decode(&balanceDoc)

			fmt.Println(balanceDoc.Amount)

			if err != nil {
				if err == mongo.ErrNoDocuments {
					fmt.Println("Balance document not found, creating new document with fixed ID...")

				}
				// Log the error (optional)

				// Create a new balance document with the fixed ID
				balanceDoc = models.Balance{
					ID:        fixedID, // Use the fixed ID
					Amount:    "0",     // Initial amount (modify as needed)
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				_, err := database.InsertDocument(models.Collection.Balance, utils.ConvertStructPrimitive(balanceDoc))

				if err != nil {
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create balance document: " + err.Error()})
					return
				}

				// Return the newly created balance document
				context.JSON(http.StatusCreated, gin.H{"balance": balanceDoc})
				return
			}

			// If the document exists, return the balance document
			context.JSON(http.StatusOK, gin.H{"balance": balanceDoc})
		})

		// POST /balance
		balanceRoutes.POST("/", func(context *gin.Context) {

			balanceDocumentID := os.Getenv("BALANCE_ID")

			auth, _ := context.Get("auth")
			authentication := auth.(*utils.Authentication)

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
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ObjectID format"})
				return
			}

			body := newFundStruct(objectId)

			// Bind the JSON request to the fund struct
			if err := context.ShouldBindJSON(&body); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to bind JSON: " + err.Error()})
				return
			}

			// Validate the required fields
			if err := models.ValidateStruct.Struct(body); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Validation error: " + err.Error()})
				return
			}

			// Insert the new fund document
			insertResult, err := database.InsertDocument(models.Collection.Fund, utils.ConvertStructPrimitive(body))
			if err != nil {
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert fund document: " + err.Error()})
				return
			}

			body.ID = insertResult.InsertedID.(primitive.ObjectID)

			// Find the balance document by ID
			var balanceDoc models.Balance
			docResult := database.FindDocumentById(models.Collection.Balance, balanceDocumentID)
			if err := docResult.Decode(&balanceDoc); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decode balance document: " + err.Error()})
				return
			}

			// Parse the amounts from strings to floats
			newAmount, err := strconv.ParseFloat(body.Amount, 64)
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fund amount format"})
				return
			}

			balanceAmount, err := strconv.ParseFloat(balanceDoc.Amount, 64)
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid balance amount format"})
				return
			}
			balanceObjectId, _ := primitive.ObjectIDFromHex(balanceDocumentID)

			s := fmt.Sprintf("%f", balanceAmount+newAmount)

			// Update the balance document by adding the new fund amount
			_, err = database.UpdateDocument(
				models.Collection.Balance,
				bson.D{{Key: "_id", Value: balanceObjectId}},
				bson.D{{Key: "$set", Value: bson.D{{Key: "amount", Value: s}}}},
			)

			if err != nil {
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update balance: " + err.Error()})
				return
			}

			// Return the updated balance
			context.JSON(http.StatusOK, gin.H{"balance": balanceAmount + newAmount})
		})
	}
}
