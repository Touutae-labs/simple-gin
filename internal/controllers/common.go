package controllers

import "github.com/gin-gonic/gin"

// Data is the success envelope's data payload.
type Data struct {
	Data1 *string `json:"data1,omitempty" example:"e5b1..."`
	Data2 *string `json:"data2,omitempty" example:"Espresso Beans"`
}

// successPayload is the shape of every 2xx response.
type successPayload struct {
	Successful bool   `json:"successful" example:"true"`
	ErrorCode  string `json:"error_code" example:""`
	Data       *Data  `json:"data" swaggertype:"object"`
}

func writeCreated(c *gin.Context, data1, data2 string) {
	c.JSON(201, successPayload{Successful: true, Data: &Data{Data1: &data1, Data2: &data2}})
}

func writeOK(c *gin.Context) {
	c.JSON(200, successPayload{Successful: true})
}
