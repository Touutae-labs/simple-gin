package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Touutae-labs/simple-gin/internal/controllers"
)

// Handler wires controllers onto the Gin engine.
type Handler struct {
	Controllers *controllers.Controllers
}

func newHandler(c *controllers.Controllers) *Handler { return &Handler{Controllers: c} }

func (h *Handler) register(app *gin.Engine) {
	// GET / redirects to the API docs — humans hitting the root
	// should land on Swagger, not a 404.
	app.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/api-docs/index.html")
	})
	app.GET("/healthz", h.Controllers.HealthController.Health)

	products := app.Group("/product")
	{
		products.GET("", h.Controllers.ProductController.List)
		products.POST("", h.Controllers.ProductController.Create)
		products.GET("/:id", h.Controllers.ProductController.Get)
		products.PATCH("/:id", h.Controllers.ProductController.Patch)
		products.DELETE("/:id", h.Controllers.ProductController.Delete)
	}
}
