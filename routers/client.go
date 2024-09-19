package routers

import (
	"errors"
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
	"go.mongodb.org/mongo-driver/mongo"
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

	clientsRoutes := router.Group("/clients")
	clientsRoutes.Use(authService.AuthMiddleware())
	{

		clientsRoutes.GET("/", func(c *gin.Context) {

			pageParam := c.Query("page")

			pageSize := 10
			page := 1

			if pageParam != "" {
				page, _ = strconv.Atoi(pageParam)
			}

			offset := (page - 1) * pageSize

			cursor, err := database.FindDocumentsQuery(models.Collection.Customer, bson.D{}, pageSize, offset)

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

		})

		clientsRoutes.GET("/search", func(c *gin.Context) {

			pageParam := c.Query("page")
			searchQuery := c.Query("q")

			pageSize := 10
			page := 1

			if pageParam != "" {
				page, _ = strconv.Atoi(pageParam)
			}

			offset := (page - 1) * pageSize

			// Create a filter based on the search query
			var filter bson.D
			if searchQuery != "" {
				filter = bson.D{
					{Key: "$or", Value: bson.A{
						bson.D{{Key: "name", Value: bson.M{"$regex": searchQuery, "$options": "i"}}},  // Case-insensitive regex search
						bson.D{{Key: "email", Value: bson.M{"$regex": searchQuery, "$options": "i"}}}, // Example field: email
						bson.D{{Key: "phone", Value: bson.M{"$regex": searchQuery, "$options": "i"}}}, // Example field: phone
					}},
				}
			}

			cursor, err := database.FindDocumentsQuery(models.Collection.Customer, filter, pageSize, offset)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": err.Error(),
				})
				c.Abort()
				return
			}

			var customers []models.Customer

			err = cursor.All(c, &customers)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": err.Error(),
				})
				c.Abort()
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"customers": customers,
				"page":      page,
			})
		})

		clientsRoutes.POST("/", func(c *gin.Context) {

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

		clientsRoutes.GET("/verify/:phone", func(c *gin.Context) {
			phone := c.Param("phone")
			if phone == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "please provide a valid phone number"})
				c.Abort()
				return
			}
			var existingCustomer models.Customer

			customerCursor := database.FindDocument(models.Collection.Customer, bson.D{{Key: "phone", Value: c.Param("phone")}})

			if err := customerCursor.Decode(&existingCustomer); err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					c.JSON(http.StatusOK, gin.H{"error": "Customer not found", "customer": nil})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
				}
				c.Abort()
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"customer": existingCustomer,
				"message":  "Existing customer found",
			})

		})

		clientsRoutes.GET("/:id", func(c *gin.Context) {
			customerId := c.Param("id")

			objID, err := primitive.ObjectIDFromHex(customerId)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer_id"})
				c.Abort()
				return
			}

			var customer models.Customer

			customerCursor := database.FindDocumentById(models.Collection.Customer, customerId)

			err = customerCursor.Decode(&customer)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer_id"})
				c.Abort()
				return
			}

			var transactions []models.Transaction
			documents, err := database.FindDocuments(models.Collection.Customer, bson.D{{Key: "customer_id", Value: objID}})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer_id"})
				c.Abort()
				return

			}
			err = documents.All(c, &transactions)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer_id"})
				c.Abort()
				return
			}

			//	get loans
			var loans []models.Loan
			loansCursor, err := database.FindDocuments(models.Collection.Loan, bson.D{{Key: "customer_id", Value: objID}})

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer_id"})
				c.Abort()
				return

			}
			err = loansCursor.All(c, &loans)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer_id"})
				c.Abort()
				return
			}

			//	now do calculations
			var totalTransactionAmount, totalLoanAmount, totalLoanSettlement float64

			for _, transaction := range transactions {
				amount, _ := strconv.ParseFloat(transaction.Amount, 64)
				totalTransactionAmount += amount
			}

			for _, loan := range loans {
				amount, _ := strconv.ParseFloat(loan.Amount, 64)
				if loan.Type == "credit" { // Assuming "credit" means loan taken
					totalLoanAmount += amount
				} else if loan.Type == "debit" { // Assuming "debit" means loan settled
					totalLoanSettlement += amount
				}
			}

			totalLoanToBePaid := totalLoanAmount - totalLoanSettlement

			// Respond with the results
			c.JSON(http.StatusOK, gin.H{
				"customer":                 customer,
				"transactions":             transactions,
				"loans":                    loans,
				"total_transaction_amount": totalTransactionAmount,
				"total_loan_amount":        totalLoanAmount,
				"total_loan_settlement":    totalLoanSettlement,
				"total_loan_to_be_paid":    totalLoanToBePaid,
			})

		})

	}

}
