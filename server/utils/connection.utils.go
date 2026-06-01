package utils

import (
	"fmt"
	"manitor-server/types"

	"gorm.io/gorm"
)

func GetParametrizedQueryInstance(query *gorm.DB, params types.GetConnectionsQueryParamsDTO) (resultQuery *gorm.DB, err error) {
	if params.WiFiName != "" {
		query = query.Where("wifi_name = ?", params.WiFiName)
	}
	if params.HostName != "" {
		query = query.Where("host_name = ?", params.HostName)
	}

	if params.TotalDownload != "desc" && params.TotalDownload != "asc" {
		return nil, fmt.Errorf("invalid total_download value: %s", params.TotalDownload)
	}
	if params.TotalUpload != "desc" && params.TotalUpload != "asc" {
		return nil, fmt.Errorf("invalid total_upload value: %s", params.TotalUpload)
	}

	if params.TotalDownload == "" {
		query = query.Order(fmt.Sprintf("total_download %s", "asc"))
	} else {
		query = query.Order(fmt.Sprintf("total_download %s", params.TotalDownload))
	}
	if params.TotalUpload == "" {
		query = query.Order(fmt.Sprintf("total_upload %s", "asc"))
	} else {
		query = query.Order(fmt.Sprintf("total_upload %s", params.TotalUpload))
	}

	if params.Page != 0 {
		query.Offset(params.Page)
	} else {
		query.Offset(1)
	}

	if params.PageSize != 0 {
		query.Limit(params.PageSize)
	} else {
		query.Limit(10)
	}

	return query, nil
}
