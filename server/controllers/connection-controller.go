package controllers

import (
	"errors"
	"fmt"
	"manitor-server/models"
	"manitor-server/repository"
	"manitor-server/types"
	"manitor-server/utils"
	"net/http"
	"strconv"

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
	var requestBody CreateControllerBodyDTO
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

	var createBody = models.Connection{
		IP:           requestBody.SystemIP,
		WiFiName:     wifiName,
		HostName:     hostName,
		DownloadSize: downloadSize,
		UploadSize:   uploadSize,
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
	intPage, pageErr := strconv.Atoi(requestContext.DefaultQuery("page", "1"))
	intPageSize, pageSizeErr := strconv.Atoi(requestContext.DefaultQuery("pageSize", "10"))

	if pageErr != nil && pageSizeErr != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(
				errors.New("Error in parsing parameters"),
			),
		)
		return
	}

	params := types.QueryParameters{
		Page:     intPage,
		PageSize: intPageSize,
	}

	connections, err := repository.GetConnections(requestContext, params)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadGateway,
			utils.ResolveError(pageErr),
		)
		return
	}

	requestContext.JSON(
		http.StatusOK,
		utils.ResolveResponse(connections),
	)
}
