package controllers

import (
	"fmt"
	"manitor-server/models"
	"manitor-server/repository"
	"manitor-server/types"
	"manitor-server/utils"
	"net/http"
	"time"

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

func CreateConnection(requestContext *gin.Context) {
	var requestBody types.CreateConnectionBodyDTO
	err := requestContext.Bind(&requestBody)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
		return
	}

	validator := utils.GetValidator()
	err = validator.Struct(requestBody)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
		return
	}

	uploadSize, downloadSize := utils.GetTransferSizes(requestBody.Logs)
	fmt.Println("سگ")
	fmt.Println(requestBody.WiFiName)
	fmt.Println(requestBody.HostName)
	wifiName := utils.GetUnknownIfEmpty(requestBody.WiFiName)
	hostName := utils.GetUnknownIfEmpty(requestBody.HostName)
	collectedAt, err := time.Parse(time.RFC3339, requestBody.CollectedAt)

	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
		return
	}

	var createBody = models.Connection{
		IP:           requestBody.SystemIP,
		WiFiName:     wifiName,
		HostName:     hostName,
		DownloadSize: downloadSize,
		UploadSize:   uploadSize,
		CollectedAt:  collectedAt,
	}
	err = repository.CreateConnection(&createBody)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
		return
	}

	requestContext.JSON(
		http.StatusOK,
		createBody,
	)
}

func HandleSessionStreamSocket(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func GetConnections(requestContext *gin.Context) {

	var queryParams types.GetConnectionsQueryParamsDTO
	err := requestContext.BindQuery(&queryParams)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
		return
	}

	connections, err := repository.GetConnections(requestContext, queryParams)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadGateway,
			utils.ResolveError(err),
		)
		return
	}

	requestContext.JSON(
		http.StatusOK,
		utils.ResolveResponse(connections),
	)
}
