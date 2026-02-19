package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/sql"
)

func sqlWarehouseFixUps(warehouse *sql.GetWarehouseResponse, userName string) {
	if warehouse.CreatorName == "" {
		warehouse.CreatorName = userName
	}

	if warehouse.MinNumClusters == 0 {
		warehouse.MinNumClusters = 1
		warehouse.ForceSendFields = append(warehouse.ForceSendFields, "MinNumClusters")
	}

	if warehouse.WarehouseType == "" {
		warehouse.WarehouseType = sql.GetWarehouseResponseWarehouseTypeClassic
	}
}

func (s *FakeWorkspace) SqlWarehousesUpsert(req Request, warehouseId string) Response {
	var warehouse sql.GetWarehouseResponse

	if err := json.Unmarshal(req.Body, &warehouse); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	defer s.LockUnlock()()

	if warehouseId != "" {
		_, ok := s.SqlWarehouses[warehouseId]
		if !ok {
			return Response{
				StatusCode: 404,
			}
		}
	} else {
		warehouseId = nextUUID()
	}
	warehouse.Id = warehouseId
	if warehouse.Name == "" {
		warehouse.Name = warehouseId
	}
	warehouse.State = sql.StateRunning
	sqlWarehouseFixUps(&warehouse, s.CurrentUser().UserName)
	s.SqlWarehouses[warehouseId] = warehouse

	return Response{
		StatusCode: 200,
		Body:       warehouse,
	}
}

func (s *FakeWorkspace) SqlWarehousesList(req Request) Response {
	var warehouses []sql.EndpointInfo
	for _, warehouse := range s.SqlWarehouses {
		warehouses = append(warehouses, sql.EndpointInfo{
			Id:   warehouse.Id,
			Name: warehouse.Name,
		})
	}
	return Response{
		StatusCode: 200,
		Body: sql.ListWarehousesResponse{
			Warehouses: warehouses,
		},
	}
}

func (s *FakeWorkspace) SqlDataSourcesList(req Request) Response {
	var dataSources []sql.DataSource
	for key, warehouse := range s.SqlWarehouses {
		dataSources = append(dataSources, sql.DataSource{
			Id:          key,
			Name:        "test_data_source",
			WarehouseId: warehouse.Id,
		})
	}
	return Response{
		StatusCode: 200,
		Body:       dataSources,
	}
}
