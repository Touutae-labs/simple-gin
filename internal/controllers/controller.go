// Package controllers holds the Gin handlers. Each controller is a
// thin struct that depends on the service interface and returns the
// {successful, error_code, data} envelope.
package controllers

import "github.com/gin-gonic/gin"

// Controllers groups every controller in the process. Built by Wire.
type Controllers struct {
	HealthController  *HealthController
	ProductController *ProductController
}

func NewControllers(health *HealthController, product *ProductController) *Controllers {
	return &Controllers{
		HealthController:  health,
		ProductController: product,
	}
}

// errorPayload is the shape of every non-2xx response.
type errorPayload struct {
	Successful bool   `json:"successful" example:"false"`
	ErrorCode  string `json:"error_code" example:"INVALID_NAME"`
	Data       any    `json:"data" swaggertype:"object"`
}

func writeError(c *gin.Context, status int, code string) {
	c.JSON(status, errorPayload{Successful: false, ErrorCode: code, Data: nil})
}

func writeServerError(c *gin.Context) {
	c.JSON(500, errorPayload{Successful: false, ErrorCode: "INTERNAL_ERROR", Data: nil})
}
