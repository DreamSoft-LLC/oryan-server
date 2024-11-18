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
	"golang.org/x/net/context"

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
			searchTerm := context.Query("q") // Add the search term
			// auth, _ := context.Get("auth")
			// authentication := auth.(*utils.Authentication)
			pageSize := 100000
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
			// 	// Assuming there is an associate_id field in your document
			// 	filter = append(filter, bson.E{Key: "associate_id", Value: authentication.ID})
			// }

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

			// Add search term to filter (Assuming you are searching a field like 'description' or 'title')
			if searchTerm != "" {
				searchFilter := bson.M{
					"$or": []bson.M{
						{"amount": bson.M{"$regex": searchTerm, "$options": "i"}}, // case-insensitive search
						{"mineral": bson.M{"$regex": searchTerm, "$options": "i"}},
						{"scale": bson.M{"$regex": searchTerm, "$options": "i"}},
					},
				}
				filter = append(filter, bson.E{Key: "$and", Value: bson.A{searchFilter}})
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

			fixedID, _ := primitive.ObjectIDFromHex(balanceDocumentID)

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

			var balanceData models.Balance
			balanceDocument := database.FindDocument(models.Collection.Balance, bson.D{{Key: "_id", Value: fixedID}})

			err = balanceDocument.Decode(&balanceData)

			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

			balanceTotalalAmountBalance, _ := strconv.ParseFloat(balanceData.Amount, 64)
			transactionAmount, _ := strconv.ParseFloat(newtransaction.Amount, 64)

			if transactionAmount > balanceTotalalAmountBalance {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Insuficient balance available contact admin"})
				return
			}

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

			balanceAmountSpent, _ := strconv.ParseFloat(balanceData.Spent, 64)

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

		transactionRoutes.GET("/profit", func(ctx *gin.Context) {
			// get all transactions made every day , and  make math operation on the transcation.type "buy" and "sell" to find the profit for eact day and return profit data for each day for a month duration

			now := time.Now()
			startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

			// Define a MongoDB filter to get transactions from the past month
			filter := bson.M{
				"created_at": bson.M{
					"$gte": startOfMonth,
					"$lte": now,
				},
			}

			var transactions []models.Transaction

			cursor, err := database.FindManyDocuments(models.Collection.Transaction, filter, bson.D{{Key: "created_at", Value: 1}})
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
				return
			}

			if err := cursor.All(context.TODO(), &transactions); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse transactions"})
				return
			}

			// Initialize a map to store profit per day
			profitPerDay := make(map[string]float64)
			var bbTransactionProfitTotal, miniTransactionProfitTotal float64

			// Loop over the transactions and calculate the daily profit
			for _, transaction := range transactions {
				// Format the date to group by day
				day := transaction.CreatedAt.Format("2006-01-02")

				// Convert the amount to float
				amount, err := strconv.ParseFloat(transaction.Amount, 64)

				if err != nil {
					// Skip invalid transactions
					continue
				}

				// weight, err := strconv.ParseFloat(transaction.Weight, 64)

				// if err != nil {
				// 	// Skip invalid transactions
				// 	continue
				// }

				// rate, err := strconv.ParseFloat(transaction.Rate, 64)

				// if err != nil {
				// 	// Skip invalid transactions
				// 	continue
				// }

				if transaction.Kind == "buy" {

					if transaction.Scale == "bb" {
						// add 4.60 to weight
						bbTransactionProfitTotal += (amount * (15.81 / 100))

					} else if transaction.Scale == "mini" {
						// add 2.70 to weight
						miniTransactionProfitTotal += (amount * (9.28 / 100))
					}

					profitPerDay[day] -= amount

				} else if transaction.Kind == "sell" {

					profitPerDay[day] += amount

				}

			}

			ctx.JSON(http.StatusOK, gin.H{"profit_per_day": profitPerDay, "bb_profit": bbTransactionProfitTotal, "mini_profit": miniTransactionProfitTotal, "gb_profit": 0.00})

		})

		transactionRoutes.GET("/profit/filter", func(ctx *gin.Context) {
			// get all transactions made every day , and  make math operation on the transcation.type "buy" and "sell" to find the profit for eact day and return profit data for each day for a month duration
			filterParam := ctx.Query("filter")

			var filter = bson.M{}

			now := time.Now()
			if filterParam != "" {
				switch filterParam {
				case "today":
					startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
					filter = bson.M{"created_at": bson.M{"$gte": startOfDay, "$lte": now}}
				case "week":
					startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
					filter = bson.M{"created_at": bson.M{"$gte": startOfWeek, "$lte": now}}
				case "month":
					startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
					filter = bson.M{"created_at": bson.M{"$gte": startOfMonth, "$lte": now}}
				case "year":
					startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
					filter = bson.M{"created_at": bson.M{"$gte": startOfYear, "$lte": now}}
				}
			}

			var transactions []models.Transaction

			cursor, err := database.FindManyDocuments(models.Collection.Transaction, filter, bson.D{{Key: "created_at", Value: 1}})
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
				return
			}

			if err := cursor.All(context.TODO(), &transactions); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse transactions"})
				return
			}

			var bbTransactionProfitTotal, miniTransactionProfitTotal float64

			// Loop over the transactions and calculate the daily profit
			for _, transaction := range transactions {

				// Convert the amount to float
				amount, err := strconv.ParseFloat(transaction.Amount, 64)

				if err != nil {
					// Skip invalid transactions
					continue
				}

				// weight, err := strconv.ParseFloat(transaction.Weight, 64)

				// if err != nil {
				// 	// Skip invalid transactions
				// 	continue
				// }

				// rate, err := strconv.ParseFloat(transaction.Rate, 64)

				// if err != nil {
				// 	// Skip invalid transactions
				// 	continue
				// }

				if transaction.Kind == "buy" {

					if transaction.Scale == "bb" {
						// add 4.60 to weight
						bbTransactionProfitTotal += (amount * (15.81 / 100))

					} else if transaction.Scale == "mini" {
						// add 2.70 to weight
						miniTransactionProfitTotal += (amount * (9.28 / 100))
					}

				}

			}

			ctx.JSON(http.StatusOK, gin.H{"bb_profit": bbTransactionProfitTotal, "mini_profit": miniTransactionProfitTotal, "gb_profit": 0.00})

		})
	}
}
