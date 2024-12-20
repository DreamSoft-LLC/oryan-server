package routers

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true, // Allows all origins
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Setup routes from other files
	SetupAuthRoutes(router)
	SetupAssociatesRoutes(router)
	SetupTransactionRoutes(router)
	SetupClientRoutes(router)
	SetupLoanRoutes(router)
	SetupBalancesRoutes(router)
	SetupMiscellaneousRoutes(router)
	SetupStashRoutes(router)
	return router
}
