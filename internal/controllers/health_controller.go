package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthController struct{}

func NewHealthController() *HealthController { return &HealthController{} }

// Health godoc
// @Summary  Liveness probe
// @Tags     Health
// @Produce  json
// @Success  200  {object}  successPayload
// @Router   /healthz [get]
func (c *HealthController) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, successPayload{Successful: true})
}
