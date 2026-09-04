package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Touutae-labs/simple-gin/internal/controllers"
)

type handler struct {
	controllers *controllers.Controllers
}


func newHandler(c *controllers.Controllers) *handler { return &handler{controllers: c} }

func (h *handler) register(app *gin.Engine) {
	app.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/api-docs/index.html")
	})
	app.GET("/healthz", h.controllers.HealthController.Health)

	products := app.Group("/product")
	{
		products.GET("", h.controllers.ProductController.List)
		products.POST("", h.controllers.ProductController.Create)
		products.GET("/:id", h.controllers.ProductController.Get)
		products.PATCH("/:id", h.controllers.ProductController.Patch)
		products.DELETE("/:id", h.controllers.ProductController.Delete)
	}
}
