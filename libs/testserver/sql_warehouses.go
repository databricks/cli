package testserver

import (
	"encoding/json"
	"fmt"
	warehouses "github.com/databricks/cli/libs/sdktypes/warehouses/v1"
	"net/http"
)

func sqlWarehouseFixUps(warehouse *warehouses.GetWarehouseResponse, userName string) {
	if val(warehouse.CreatorName) == "" {
		warehouse.CreatorName = nonZero(userName)
	}

	if val(warehouse.MinNumClusters) == 0 {
		warehouse.MinNumClusters = new(1)
	}

	if warehouse.WarehouseType == "" {
		warehouse.WarehouseType = warehouses.WarehouseType_Classic
	}
}

func (s *FakeWorkspace) SqlWarehousesUpsert(req Request, warehouseId string) Response {
	var warehouse warehouses.GetWarehouseResponse

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
	warehouse.Id = nonZero(warehouseId)
	if val(warehouse.Name) == "" {
		warehouse.Name = nonZero(warehouseId)
	}
	warehouse.State = warehouses.EndpointState_Running
	sqlWarehouseFixUps(&warehouse, val(s.CurrentUser().UserName))
	s.SqlWarehouses[warehouseId] = warehouse

	return Response{
		StatusCode: 200,
		Body:       warehouse,
	}
}

func (s *FakeWorkspace) SqlWarehousesStart(req Request, warehouseId string) Response {
	defer s.LockUnlock()()

	warehouse, ok := s.SqlWarehouses[warehouseId]
	if !ok {
		return Response{StatusCode: 404}
	}

	warehouse.State = warehouses.EndpointState_Running
	s.SqlWarehouses[warehouseId] = warehouse

	return Response{}
}

func (s *FakeWorkspace) SqlWarehousesStop(req Request, warehouseId string) Response {
	defer s.LockUnlock()()

	warehouse, ok := s.SqlWarehouses[warehouseId]
	if !ok {
		return Response{StatusCode: 404}
	}

	warehouse.State = warehouses.EndpointState_Stopped
	s.SqlWarehouses[warehouseId] = warehouse

	return Response{}
}

func (s *FakeWorkspace) SqlWarehousesList(req Request) Response {
	var list []warehouses.EndpointInfo
	for _, warehouse := range s.SqlWarehouses {
		list = append(list, warehouses.EndpointInfo{
			Id:   warehouse.Id,
			Name: warehouse.Name,
		})
	}
	return Response{
		StatusCode: 200,
		Body: warehouses.ListWarehousesResponse{
			Warehouses: list,
		},
	}
}

func (s *FakeWorkspace) SqlDataSourcesList(req Request) Response {
	var dataSources []sqlDataSource
	for key, warehouse := range s.SqlWarehouses {
		dataSources = append(dataSources, sqlDataSource{
			Id:          key,
			Name:        "test_data_source",
			WarehouseId: val(warehouse.Id),
		})
	}
	return Response{
		StatusCode: 200,
		Body:       dataSources,
	}
}

// sqlDataSource is an entry of the legacy /api/2.0/preview/sql/data_sources
// response, which has no sdk-go type.
type sqlDataSource struct {
	Id          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	WarehouseId string `json:"warehouse_id,omitempty"`
}
