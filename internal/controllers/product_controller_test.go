// Controller-shape tests for ProductController. The service is
// replaced with a mockery-generated MockService, so these tests
// run with no DB and no real product.Service — the controller's
// behaviour in isolation is what's under test.
//
// Follows the POS pattern: the controller depends on a
// product.Service interface, and the generated testify mock
// implements that interface for tests that need to assert exact
// call sequences (Create was called once with these args, etc.).
package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Touutae-labs/simple-gin/internal/controllers"
	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	productmock "github.com/Touutae-labs/simple-gin/internal/mocks/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
)

func init() { gin.SetMode(gin.TestMode) }

// newTestApp wires a router with the controller under test. Only
// the routes the tests exercise are mounted; we don't pull in
// di.Initialize because the whole point of the mock is to skip
// the full composition graph.
func newTestApp(svc product.Service) *gin.Engine {
	app := gin.New()
	pc := controllers.NewProductController(svc)
	app.POST("/product", pc.Create)
	app.GET("/product/:id", pc.Get)
	return app
}


func TestCreateProduct_CallsServiceOnce(t *testing.T) {
	mockSvc := productmock.NewMockService(t)
	mockSvc.EXPECT().
		Create(anyCtx(), anyInputMatching("Espresso")).
		Return(models.Result{ProductID: "abc-123", Name: "Espresso"}, nil).
		Once()

	app := newTestApp(mockSvc)
	body := bytes.NewBufferString(`{"name":"Espresso","description":"dark","price":29.90}`)
	req := httptest.NewRequest(http.MethodPost, "/product", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Equal(t, true, out["successful"])

	// mockery's t.Cleanup AssertExpectations runs at test end.
}


func TestGetProduct_NotFoundFromService(t *testing.T) {
	mockSvc := productmock.NewMockService(t)
	mockSvc.EXPECT().
		Get(anyCtx(), "missing-id").
		Return(nil, &models.Error{Code: models.CodeProductNotFound, Field: "id", Message: "missing-id not found"}).
		Once()

	app := newTestApp(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/product/missing-id", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "PRODUCT_NOT_FOUND")
}


func TestGetProduct_ServiceErrorMappedTo500(t *testing.T) {
	mockSvc := productmock.NewMockService(t)
	mockSvc.EXPECT().
		Get(anyCtx(), "any").
		Return(nil, &models.Error{Code: models.CodeRepositoryFailure, Message: "db down"}).
		Once()

	app := newTestApp(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/product/any", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}


func stringPtr(s string) *string { return &s }

// anyCtx matches any context.Context. testify v1.11's
// AnythingOfType uses reflect.TypeOf(...).String() to compare,
// which fails for interface types (returns the concrete
// type, e.g. *context.emptyCtx, not the interface name).
// MatchedBy runs a predicate instead, so the interface check works.
func anyCtx() interface{} {
	return mock.MatchedBy(func(_ context.Context) bool { return true })
}


// anyInputMatching matches a CreateInput whose Name equals want.
// testify does pointer/value comparison on struct fields, so
// checking the whole struct fails on the *string Description
// and *big.Int Price. Match the only field the test cares about.
func anyInputMatching(want string) interface{} {
	return mock.MatchedBy(func(in models.CreateInput) bool { return in.Name == want })
}
