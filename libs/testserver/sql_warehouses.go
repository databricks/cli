package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	warehouses "github.com/databricks/cli/libs/sdktypes/warehouses/v1"
	"github.com/databricks/cli/libs/sqllegacy"
	"github.com/databricks/cli/libs/utils"
)

func sqlWarehouseFixUps(warehouse *warehouses.GetWarehouseResponse, userName string) {
	if utils.Deref(warehouse.CreatorName) == "" {
		warehouse.CreatorName = utils.NonZero(userName)
	}

	if utils.Deref(warehouse.MinNumClusters) == 0 {
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
	warehouse.Id = utils.NonZero(warehouseId)
	if utils.Deref(warehouse.Name) == "" {
		warehouse.Name = utils.NonZero(warehouseId)
	}
	warehouse.State = warehouses.EndpointState_Running
	sqlWarehouseFixUps(&warehouse, utils.Deref(s.CurrentUser().UserName))
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
	var dataSources []sqllegacy.DataSource
	for key, warehouse := range s.SqlWarehouses {
		dataSources = append(dataSources, sqllegacy.DataSource{
			Id:          key,
			Name:        "test_data_source",
			WarehouseId: utils.Deref(warehouse.Id),
		})
	}
	return Response{
		StatusCode: 200,
		Body:       dataSources,
	}
}
