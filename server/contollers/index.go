package controllers

import (
	"manitor-server/types"
	"manitor-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleHealth(requestContext *gin.Context) {
	requestContext.JSON(
		http.StatusOK,
		gin.H{
			"message": "System Operational!",
		},
	)
}

func HandleIngest(requestContext *gin.Context) {
	var requestBody types.AgentPayload
	err := requestContext.BindJSON(requestBody)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
	}

	validator := utils.GetValidator()
	err = validator.Struct(requestBody)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
	}

	// uploadSize, downloadSize := utils.GetTransferSizes(requestBody.Logs)

}

func HandleSessionStreamSocket(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func HandleConnections(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
