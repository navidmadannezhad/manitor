package utils

import (
	"fmt"
	"manitor-server/models"
	"manitor-server/types"

	"gorm.io/gorm"
)

func GetParametrizedQueryInstance(query *gorm.DB, params types.GetConnectionsQueryParamsDTO) (*gorm.DB, error) {
	if params.WifiName != "" {
		query = query.Where("wifi_name = ?", params.WifiName)
	}
	if params.HostName != "" {
		query = query.Where("host_name = ?", params.HostName)
	}
	if params.AfterID != nil {
		query = query.Where("id > ?", *params.AfterID)
	}

	if params.TotalDownload != "" {
		if params.TotalDownload != "asc" && params.TotalDownload != "desc" {
			return nil, fmt.Errorf("invalid total_download value: %s (must be 'asc' or 'desc')", params.TotalDownload)
		}
		query = query.Order(fmt.Sprintf("total_download %s", params.TotalDownload))
	}

	if params.TotalUpload != "" {
		if params.TotalUpload != "asc" && params.TotalUpload != "desc" {
			return nil, fmt.Errorf("invalid total_upload value: %s (must be 'asc' or 'desc')", params.TotalUpload)
		}
		query = query.Order(fmt.Sprintf("total_upload %s", params.TotalUpload))
	}

	if params.TotalDownload == "" && params.TotalUpload == "" {
		query = query.Order("id DESC")
	}

	page := params.Page
	if page < 1 {
		page = 1
	}

	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	return query, nil
}

func ConnectionToConnectionResponseDTO(c *models.Connection) types.ConnectionResponseDTO {
	return types.ConnectionResponseDTO{
		ID:            c.ID,
		IP:            c.IP,
		WifiName:      c.WifiName,
		HostName:      c.HostName,
		DownloadSize:  c.DownloadSize,
		UploadSize:    c.UploadSize,
		TotalDownload: c.TotalDownload,
		TotalUpload:   c.TotalUpload,
		CollectedAt:   c.CollectedAt.String(),
	}
}

func ConnectionGroupToConnectionResponseDTOGroup(group *[]models.Connection) []types.ConnectionResponseDTO {
	var result []types.ConnectionResponseDTO
	for _, c := range *group {
		result = append(result, ConnectionToConnectionResponseDTO(&c))
	}

	return result
}
