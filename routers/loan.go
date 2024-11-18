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

func newLoanStruct(associate primitive.ObjectID) *models.Loan {
	return &models.Loan{
		AssociateID: associate,
		ID:          primitive.NewObjectID(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func SetupLoanRoutes(router *gin.Engine) {
	jwtAuthService := utils.GetJWTAuthService()
	loanRoutes := router.Group("/loans")
	loanRoutes.Use(jwtAuthService.AuthMiddleware())
	{

		loanRoutes.GET("", func(context *gin.Context) {
			filterParam := context.Query("filter")
			pageParam := context.Query("page")
			// auth, _ := context.Get("auth")
			// authentication := auth.(*utils.Authentication)
			pageSize := 1000000
			page := 1

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

			loans_, err := database.GetAllLoansWithAssociatesAndCustomers(filter, pageSize, offset)

			if err != nil {
				context.JSON(http.StatusOK, gin.H{
					"message": err.Error(),
				})
				context.Abort()
				return
			}

			context.JSON(http.StatusOK, gin.H{
				"loans": loans_,
				"page":  page,
			})

			//cursor, err := database.FindDocumentsQuery(models.Collection.Loan, filter, pageSize, offset)
			//
			//if err != nil {
			//	context.JSON(http.StatusOK, gin.H{
			//		"message": err.Error(),
			//	})
			//	context.Abort()
			//	return
			//}
			//
			//var loans []models.Loan
			//
			//err = cursor.All(context, &loans)
			//
			//if err != nil {
			//	context.JSON(http.StatusOK, gin.H{
			//		"message": err.Error(),
			//	})
			//	context.Abort()
			//	return
			//}
			//
			//context.JSON(http.StatusOK, gin.H{
			//	"loans": loans,
			//	"page":  page,
			//})
		})

		loanRoutes.POST("", func(context *gin.Context) {
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
			newLoan := newLoanStruct(objectId)

			if err := context.ShouldBindJSON(&newLoan); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			err = models.ValidateStruct.Struct(newLoan)

			if err != nil {
				//TODO: return an error response of the required fields left empty
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			insertResult, err := database.InsertDocument(models.Collection.Loan, utils.ConvertStructPrimitive(newLoan))

			newLoan.ID = insertResult.InsertedID.(primitive.ObjectID)

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
			loanAmount, _ := strconv.ParseFloat(newLoan.Amount, 64)

			if newLoan.Type == "credit" {
				s := fmt.Sprintf("%f", balanceAmountSpent+loanAmount)

				_, err := database.UpdateDocument(models.Collection.Balance, bson.D{{Key: "_id", Value: fixedID}}, bson.D{{Key: "$set", Value: bson.D{{Key: "spent", Value: s}}}})

				if err != nil {
					context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					context.Abort()
					return
				}

			}

			balanceAmount, _ := strconv.ParseFloat(balanceData.Amount, 64)
			if newLoan.Type == "payoff" {
				s := fmt.Sprintf("%f", balanceAmount+loanAmount)

				_, err := database.UpdateDocument(models.Collection.Balance, bson.D{{Key: "_id", Value: fixedID}}, bson.D{{Key: "$set", Value: bson.D{{Key: "amount", Value: s}}}})

				if err != nil {
					context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					context.Abort()
					return
				}

			}

			context.JSON(http.StatusOK, gin.H{
				"created": insertResult,
				"loan":    newLoan,
				"message": "Successfully added a new transaction",
			})
		})

	}

}
