package repository

import (
	"context"
	"manitor-server/infra"
	"manitor-server/models"
	"manitor-server/types"
	"manitor-server/utils"
)

func GetConnections(requestContext context.Context, params types.GetConnectionsQueryParamsDTO) ([]models.Connection, error) {
	var connections []models.Connection

	query := infra.DB.WithContext(requestContext).Model(&models.Connection{})
	parametrizedQuery, err := utils.GetParametrizedQueryInstance(query, params)
	if err != nil {
		return nil, err
	}

	result := parametrizedQuery.Find(&connections)
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
