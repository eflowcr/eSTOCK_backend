package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type LocationsController struct {
	Service services.LocationsService
}

func NewLocationsController(service services.LocationsService) *LocationsController {
	return &LocationsController{
		Service: service,
	}
}

func (c *LocationsController) GetAllLocations(ctx *gin.Context) {
	locations, response := c.Service.GetAllLocations()

	if response != nil {
		tools.Response(ctx, "GetAllLocations", false, response.Message, "get_all_locations", nil, false, "")
		return
	}

	if len(locations) == 0 {
		tools.Response(ctx, "GetAllLocations", true, "No locations found", "get_all_locations", nil, false, "")
		return
	}

	tools.Response(ctx, "GetAllLocations", true, "Locations retrieved successfully", "get_all_locations", locations, false, "")
}
