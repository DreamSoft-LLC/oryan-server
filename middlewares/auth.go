package middlewares

import (
	"github.com/DreamSoft-LLC/oryan/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func IsAdminValidate() gin.HandlerFunc {
	return func(c *gin.Context) {

		auth, exist := c.Get("auth")
		if exist == false {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "You don't have permission to access this resource",
			})
			c.Abort()
			return
		}

		authInfo, ok := auth.(*utils.Authentication)

		if !ok {
			println(authInfo)
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Invalid authentication data",
			})
			c.Abort()
			return
		}

		if authInfo.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"status": "Forbidden", "message": "You don't have permission to add an associate"})
			c.Abort()
			return
		}
		
		c.Next()

	}
}
