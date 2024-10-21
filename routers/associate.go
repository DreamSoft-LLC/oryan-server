package routers

import (
	"context"
	"net/http"
	"time"

	"github.com/DreamSoft-LLC/oryan/database"
	"github.com/DreamSoft-LLC/oryan/middlewares"
	"github.com/DreamSoft-LLC/oryan/models"
	"github.com/DreamSoft-LLC/oryan/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

			body.ID = insertResult.InsertedID.(primitive.ObjectID)

			c.JSON(http.StatusOK, gin.H{
				"created":   insertResult,
				"associate": body,
				"message":   "Successfully added a new associate",
			})

			return
		})

		//get list of associate route
		associateRoutes.GET("/", middlewares.IsAdminValidate(), func(c *gin.Context) {
			//TODO: validate is user has permission to get associate list
			var filter = bson.D{}
			searchTerm := c.Query("q")
			//get all associate
			var associates []models.Associate

			if searchTerm != "" {
				searchFilter := bson.M{
					"$or": []bson.M{
						{"name": bson.M{"$regex": searchTerm, "$options": "i"}}, // case-insensitive search
						{"email": bson.M{"$regex": searchTerm, "$options": "i"}},
						{"phone_number": bson.M{"$regex": searchTerm, "$options": "i"}},
					},
				}
				filter = append(filter, bson.E{Key: "$and", Value: bson.A{searchFilter}})
			}

			dataCursor, err := database.FindDocuments(models.Collection.Associate, filter)

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

		// get an associate record along with data depending on the tab parameter
		associateRoutes.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			var associate models.Associate

			// Fetch the associate record by ID
			docResult := database.FindDocumentById(models.Collection.Associate, id)
			err := docResult.Decode(&associate)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Associate not found"})
				c.Abort()
				return
			}

			// Get tab from the query parameter
			tab := c.Query("tab")

			// Get today's date
			startOfDay := time.Now().UTC().Truncate(24 * time.Hour)

			// Define the filter for associate_id
			associateFilter := bson.M{"associate_id": associate.ID}

			// Initialize an empty response map
			response := gin.H{
				"associate": associate,
			}

			var todayTransactionSum primitive.Decimal128
			var allTimeTransactionSum primitive.Decimal128

			err = database.SumDocuments(models.Collection.Transaction, bson.M{
				"associate_id": associate.ID,
				"created_at":   bson.M{"$gte": startOfDay},
			}, "amount", &todayTransactionSum)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate today's transaction sum", "message": err.Error()})
				c.Abort()
				return
			}

			err = database.SumDocuments(models.Collection.Transaction, associateFilter, "amount", &allTimeTransactionSum)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate all-time transaction sum", "message": err.Error()})
				c.Abort()
				return
			}

			// Sum of loans
			var todayLoanSum primitive.Decimal128
			var allTimeLoanSum primitive.Decimal128

			err = database.SumDocuments(models.Collection.Loan, bson.M{
				"associate_id": associate.ID,
				"created_at":   bson.M{"$gte": startOfDay},
			}, "amount", &todayLoanSum)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate today's loan sum"})
				c.Abort()
				return
			}

			err = database.SumDocuments(models.Collection.Loan, associateFilter, "amount", &allTimeLoanSum)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate all-time loan sum"})
				c.Abort()
				return
			}

			// Sum of miscellaneous
			var todayMiscellaneousSum primitive.Decimal128
			var allTimeMiscellaneousSum primitive.Decimal128

			err = database.SumDocuments(models.Collection.Miscellaneous, bson.M{
				"associate_id": associate.ID,
				"created_at":   bson.M{"$gte": startOfDay},
			}, "amount", &todayMiscellaneousSum)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate today's miscellaneous sum"})
				c.Abort()
				return
			}

			err = database.SumDocuments(models.Collection.Miscellaneous, associateFilter, "amount", &allTimeMiscellaneousSum)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate all-time miscellaneous sum"})
				c.Abort()
				return
			}

			// Add sums to the response
			response["today_transaction_sum"] = todayTransactionSum
			response["all_time_transaction_sum"] = allTimeTransactionSum
			response["today_loan_sum"] = todayLoanSum
			response["all_time_loan_sum"] = allTimeLoanSum
			response["today_miscellaneous_sum"] = todayMiscellaneousSum
			response["all_time_miscellaneous_sum"] = allTimeMiscellaneousSum

			// Handle different tabs to fetch the relevant records
			switch tab {
			case "transaction":
				var transactions []models.Transaction
				filter := bson.D{{Key: "associate_id", Value: associate.ID}}

				cursor, err := database.FindDocuments(models.Collection.Transaction, filter)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions", "message": err.Error()})
					c.Abort()
					return
				}

				err = cursor.All(context.TODO(), &transactions)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions", "message": err.Error()})
					c.Abort()
					return
				}

				response["transactions"] = transactions

			case "loan":
				var loans []models.Loan
				filter := bson.D{{Key: "associate_id", Value: associate.ID}}

				loanCursor, err := database.FindDocuments(models.Collection.Loan, filter)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch loans"})
					c.Abort()
					return
				}

				err = loanCursor.All(context.TODO(), &loans)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch loans"})
					c.Abort()
					return
				}

				response["loans"] = loans

			case "customer":
				var customers []models.Customer
				filter := bson.D{{Key: "created_by", Value: associate.ID}}

				customersCursor, err := database.FindDocuments(models.Collection.Customer, filter)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customers"})
					c.Abort()
					return
				}

				err = customersCursor.All(context.TODO(), &customers)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customers"})
					c.Abort()
					return
				}

				response["customers"] = customers

			case "miscellaneous":
				var miscellaneous []models.Miscellaneous
				filter := bson.D{{Key: "associate_id", Value: associate.ID}}

				miscellaneousCursor, err := database.FindDocuments(models.Collection.Miscellaneous, filter)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch miscellaneous records"})
					c.Abort()
					return
				}

				err = miscellaneousCursor.All(context.TODO(), &miscellaneous)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch miscellaneous"})
					c.Abort()
					return
				}

				response["miscellaneous"] = miscellaneous

			default:
				// If no tab or an invalid tab is provided, just return the associate data
				c.JSON(http.StatusOK, response)
				return
			}

			// Send the full response based on the selected tab
			c.JSON(http.StatusOK, response)
			return
		})

	}
}
