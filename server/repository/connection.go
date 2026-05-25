package repository

import (
	"context"
	"manitor-server/infra"
	"manitor-server/models"
	"manitor-server/types"
	"manitor-server/utils"
)

func GetConnections(requestContext context.Context, params types.QueryParameters) ([]models.Connection, error) {
	var connections []models.Connection
	result := infra.DB.WithContext(requestContext).Scopes(
		utils.Paginate(params.Page, params.PageSize),
	).Find(&connections)

	if result.Error != nil {
		return nil, result.Error
	}

	return connections, nil
}

func CreateConnection(connection *models.Connection) error {
	result := infra.DB.Create(connection)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
