package routes

import (
	controllers "manitor-server/contollers"

	"github.com/gin-gonic/gin"
)

func GetRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/", controllers.HandleIngest)
	router.GET("/health", controllers.HandleHealth)

	apiRouter := router.Group("/api/v1")
	apiRouter.GET("/connections", controllers.HandleConnections)
	apiRouter.GET("/connections/stream", controllers.HandleSessionStreamSocket)

	return router
}
