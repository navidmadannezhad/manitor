package routes

import (
	controllers "manitor-server/controllers"

	"github.com/gin-gonic/gin"
)

func GetRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/health", controllers.HandleHealth)

	apiRouter := router.Group("/api/v1")
	apiRouter.GET("/connections", controllers.GetConnections)
	apiRouter.POST("/connections", controllers.CreateConnection)
	apiRouter.GET("/connections/stream", controllers.HandleSessionStreamSocket)

	return router
}
