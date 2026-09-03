package server

import (
	"github.com/gin-gonic/gin"

	"github.com/Touutae-labs/simple-gin/internal/controllers"
)

// Handler wires controllers onto the Gin engine.
type Handler struct {
	Controllers *controllers.Controllers
}

func newHandler(c *controllers.Controllers) *Handler { return &Handler{Controllers: c} }

func (h *Handler) register(app *gin.Engine) {
	app.GET("/healthz", h.Controllers.HealthController.Health)

	products := app.Group("/product")
	{
		products.POST("", h.Controllers.ProductController.Create)
		products.PATCH("/:id", h.Controllers.ProductController.Patch)
	}
}
