package controllers

import (
	"fmt"
	"log"
	"manitor-server/config"
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
	WifiName := utils.GetUnknownIfEmpty(requestBody.WifiName)
	hostName := utils.GetUnknownIfEmpty(requestBody.HostName)
	collectedAt, err := time.Parse(time.RFC3339, requestBody.CollectedAt)

	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
		return
	}

	var prevRecordTotalDownload uint64
	var prevRecordTotalUpload uint64
	prevRecordParams := types.GetConnectionsQueryParamsDTO{}
	prevRecordParams.PageSize = 1
	connections, err := repository.GetConnections(requestContext, prevRecordParams)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
	}
	if len(connections) == 0 {
		prevRecordTotalDownload = 0
		prevRecordTotalUpload = 0
	} else {
		prevRecordTotalDownload = connections[0].TotalDownload
		prevRecordTotalUpload = connections[0].TotalUpload
	}

	var createBody = models.Connection{
		IP:            requestBody.SystemIP,
		WifiName:      WifiName,
		HostName:      hostName,
		DownloadSize:  downloadSize,
		UploadSize:    uploadSize,
		TotalDownload: prevRecordTotalDownload + downloadSize,
		TotalUpload:   prevRecordTotalUpload + uploadSize,
		CollectedAt:   collectedAt,
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
		utils.ResolveResponse(utils.ConnectionGroupToConnectionResponseDTOGroup(&connections)),
	)
}

func HandleSessionStreamSocket(requestContext *gin.Context) {
	wsUpgrader := config.GetWebsocketUpgrader()
	var params types.GetConnectionsQueryParamsDTO

	err := requestContext.BindQuery(&params)
	if err != nil {
		requestContext.AbortWithStatusJSON(
			http.StatusBadRequest,
			utils.ResolveError(err),
		)
		return
	}

	params.PageSize = 10
	connections, err := repository.GetConnections(requestContext, params)
	if err != nil {
		fmt.Println("ERROR HERE")
		fmt.Println(utils.ResolveError(err))
		requestContext.AbortWithStatusJSON(
			http.StatusInternalServerError,
			utils.ResolveError(err),
		)
		return
	}

	websocketHandler, err := wsUpgrader.Upgrade(
		requestContext.Writer,
		requestContext.Request,
		nil,
	)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer websocketHandler.Close()

	params.AfterID = &[]uint{0}[0]
	if len(connections) > 0 {
		params.AfterID = &connections[len(connections)-1].ID
	}

	err = websocketHandler.WriteJSON(map[string]interface{}{
		"type":      "history",
		"host_name": params.HostName,
		"wifi_name": params.WifiName,
		"data":      connections,
	})
	if err != nil {
		log.Printf("Failed to send initial data: %v", err)
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		newRows, err := repository.GetConnections(requestContext, params)
		if err != nil {
			_ = websocketHandler.WriteJSON(map[string]any{
				"type":    "error",
				"message": "stream query failed",
			})
			return
		}

		if len(newRows) == 0 {
			continue
		}

		params.AfterID = &newRows[len(newRows)-1].ID
		if err := websocketHandler.WriteJSON(map[string]any{
			"type":      "update",
			"host_name": params.HostName,
			"wifi_name": params.WifiName,
			"data":      newRows,
		}); err != nil {
			log.Printf("Failed to send update: %v", err)
			return
		}
	}
}

func TestSocket(requestContext *gin.Context) {
	upgrader := config.GetWebsocketUpgrader()
	socketHandler, err := upgrader.Upgrade(requestContext.Writer, requestContext.Request, nil)
	if err != nil {
		fmt.Println("234")
		requestContext.AbortWithStatusJSON(
			http.StatusInternalServerError,
			utils.ResolveError(err),
		)
		return
	}

	defer socketHandler.Close()

	for {
		fmt.Println("11")
		mt, message, err := socketHandler.ReadMessage()
		if err != nil {
			fmt.Println("fds")
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		fmt.Println("It's being called")
		test := []byte("گوز جن هستی")
		err = socketHandler.WriteMessage(mt, test)
		if err != nil {
			fmt.Println("here?")
			log.Println("write:", err)
			break
		}
	}
}
