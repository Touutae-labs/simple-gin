package controllers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
)

// CreateRequest is the POST /product body. price is required and is
// accepted as a JSON number at the HTTP boundary; the controller
// converts to shopspring/decimal before reaching the service.
type CreateRequest struct {
	Name        string   `json:"name" binding:"required" example:"Espresso Beans"`
	Description *string  `json:"description" example:"Single-origin Ethiopian roast"`
	SalePrice   *float64 `json:"sale_price" example:"24.50"`
	Price       float64  `json:"price" binding:"required" example:"29.90"`
}

// PatchRequest is the PATCH /product/{id} body. All fields optional.
// *string pointing to "" explicitly clears the field.
type PatchRequest struct {
	Name        *string  `json:"name,omitempty" example:"Updated name"`
	Description *string  `json:"description,omitempty" example:"new description"`
	SalePrice   *float64 `json:"sale_price,omitempty" example:"24.50"`
	Price       *float64 `json:"price,omitempty" example:"31.50"`
}

// ProductResponse is the JSON shape returned by GET /product and
// GET /product/{id}. Lives in the controller (not the domain) so the
// domain types stay clean of JSON tags.
type ProductResponse struct {
	ID          string   `json:"id" example:"9d95d864-87f7-4620-a3b5-b185a1536926"`
	Name        string   `json:"name" example:"Espresso Beans"`
	Description *string  `json:"description,omitempty" example:"Single-origin Ethiopian"`
	SalePrice   *float64 `json:"sale_price,omitempty" example:"24.50"`
	Price       float64  `json:"price" example:"29.90"`
	CreatedAt   string   `json:"created_at" example:"2026-09-04T02:00:00Z"`
	UpdatedAt   string   `json:"updated_at" example:"2026-09-04T02:00:00Z"`
}

// ListResponse is the JSON shape returned by GET /product. Items
// is the current page; NextCursor is "" when there are no more
// pages. The list endpoint is the only one that returns a
// paginated envelope; the others use the success envelope in
// common.go.
type ListResponse struct {
	Items      []ProductResponse `json:"items"`
	NextCursor string            `json:"next_cursor"`
	Limit      int               `json:"limit"`
}

func toResponse(p *models.Product) ProductResponse {
	r := ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price.InexactFloat64(),
		CreatedAt:   p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if p.SalePrice != nil {
		v := p.SalePrice.InexactFloat64()
		r.SalePrice = &v
	}
	return r
}

type ProductController struct {
	svc product.Service
}

func NewProductController(svc product.Service) *ProductController {
	return &ProductController{svc: svc}
}

// List godoc
// @Summary  List products
// @Tags     Products
// @Produce  json
// @Param    cursor     query  string  false  "Page cursor (id of last product from previous page). Empty = first page."
// @Param    limit      query  int     false  "Page size, max 100. 0 or omitted = default 20."
// @Param    name       query  string  false  "Case-insensitive contains-match on product name."
// @Param    min_price  query  number  false  "Inclusive lower bound on price."
// @Param    max_price  query  number  false  "Inclusive upper bound on price."
// @Success  200  {object}  ListResponse
// @Failure  422  {object}  errorPayload
// @Failure  500  {object}  errorPayload
// @Router   /product [get]
func (c *ProductController) List(ctx *gin.Context) {
	filter, perr := parseListFilter(ctx)
	if perr != nil {
		c.respondError(ctx, perr)
		return
	}
	page, perr := c.svc.List(ctx.Request.Context(), filter)
	if perr != nil {
		c.respondError(ctx, perr)
		return
	}
	out := ListResponse{
		Items:      make([]ProductResponse, len(page.Items)),
		NextCursor: page.NextCursor,
		Limit:      product.DefaultListLimit,
	}
	if filter != nil && filter.Limit > 0 {
		out.Limit = filter.Limit
	}
	for i := range page.Items {
		out.Items[i] = toResponse(&page.Items[i])
	}
	ctx.JSON(http.StatusOK, out)
}

// parseListFilter reads ?cursor= ?limit= ?name= ?min_price= ?max_price=
// and validates the shape. The service layer still validates the
// business rules (price range, limit cap) — this only handles
// parse errors. Returns the concrete *models.Error so callers can
// pass it to respondError without a type assertion.
func parseListFilter(ctx *gin.Context) (*models.ListFilter, *models.Error) {
	f := &models.ListFilter{
		Cursor: strings.TrimSpace(ctx.Query("cursor")),
	}
	if v := strings.TrimSpace(ctx.Query("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, &models.Error{
				Code:    models.CodeInvalidLimit,
				Field:   "limit",
				Message: "limit must be an integer",
			}
		}
		f.Limit = n
	}
	if v := strings.TrimSpace(ctx.Query("name")); v != "" {
		f.Name = v
	}
	if v := strings.TrimSpace(ctx.Query("min_price")); v != "" {
		d, err := decimal.NewFromString(v)
		if err != nil {
			return nil, &models.Error{
				Code:    models.CodeInvalidPriceRange,
				Field:   "min_price",
				Message: "min_price must be a decimal number",
			}
		}
		f.MinPrice = &d
	}
	if v := strings.TrimSpace(ctx.Query("max_price")); v != "" {
		d, err := decimal.NewFromString(v)
		if err != nil {
			return nil, &models.Error{
				Code:    models.CodeInvalidPriceRange,
				Field:   "max_price",
				Message: "max_price must be a decimal number",
			}
		}
		f.MaxPrice = &d
	}
	return f, nil
}

// Get godoc
// @Summary  Get a product by id
// @Tags     Products
// @Produce  json
// @Param    id  path  string  true  "Product id (UUID)"
// @Success  200  {object}  ProductResponse
// @Failure  404  {object}  errorPayload
// @Failure  500  {object}  errorPayload
// @Router   /product/{id} [get]
func (c *ProductController) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		writeError(ctx, http.StatusBadRequest, "INVALID_ID")
		return
	}
	p, perr := c.svc.Get(ctx.Request.Context(), id)
	if perr != nil {
		c.respondError(ctx, perr)
		return
	}
	ctx.JSON(http.StatusOK, toResponse(p))
}

// Create godoc
// @Summary  Create a product
// @Tags     Products
// @Accept   json
// @Produce  json
// @Param    body  body      CreateRequest  true  "Product payload"
// @Success  201   {object}  successPayload
// @Failure  400   {object}  errorPayload
// @Failure  422   {object}  errorPayload
// @Failure  500   {object}  errorPayload
// @Router   /product [post]
func (c *ProductController) Create(ctx *gin.Context) {
	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(ctx.Request.Context(), "product.create.bind_error", "err", err.Error())
		writeError(ctx, http.StatusBadRequest, "INVALID_BODY")
		return
	}
	res, perr := c.svc.Create(ctx.Request.Context(), models.CreateInput{
		Name:        req.Name,
		Description: req.Description,
		SalePrice:   floatToMoneyPtr(req.SalePrice),
		Price:       decimal.NewFromFloat(req.Price),
	})
	if perr != nil {
		c.respondError(ctx, perr)
		return
	}
	writeCreated(ctx, res.ProductID, res.Name)
}

// Patch godoc
// @Summary  Update a product (partial)
// @Tags     Products
// @Accept   json
// @Produce  json
// @Param    id    path      string         true  "Product id (UUID)"
// @Param    body  body      PatchRequest   true  "Fields to update"
// @Success  200   {object}  successPayload
// @Failure  400   {object}  errorPayload
// @Failure  404   {object}  errorPayload
// @Failure  422   {object}  errorPayload
// @Failure  500   {object}  errorPayload
// @Router   /product/{id} [patch]
func (c *ProductController) Patch(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		writeError(ctx, http.StatusBadRequest, "INVALID_ID")
		return
	}
	var req PatchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(ctx.Request.Context(), "product.patch.bind_error", "err", err.Error())
		writeError(ctx, http.StatusBadRequest, "INVALID_BODY")
		return
	}
	in := models.PatchInput{
		Name:        req.Name,
		Description: req.Description,
	}
	if req.SalePrice != nil {
		d := decimal.NewFromFloat(*req.SalePrice)
		in.SalePrice = &d
	}
	if req.Price != nil {
		d := decimal.NewFromFloat(*req.Price)
		in.Price = &d
	}
	_, perr := c.svc.Patch(ctx.Request.Context(), id, in)
	if perr != nil {
		c.respondError(ctx, perr)
		return
	}
	writeOK(ctx)
}

// Delete godoc
// @Summary  Soft-delete a product
// @Description  Marks the product as deleted. Subsequent GET /product and
// @Description  GET /product/{id} calls will return 404. Idempotent —
// @Description  deleting an already-deleted product returns 204.
// @Tags     Products
// @Produce  json
// @Param    id  path  string  true  "Product id (UUID)"
// @Success  204
// @Failure  404  {object}  errorPayload
// @Failure  500  {object}  errorPayload
// @Router   /product/{id} [delete]
func (c *ProductController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		writeError(ctx, http.StatusBadRequest, "INVALID_ID")
		return
	}
	if perr := c.svc.Delete(ctx.Request.Context(), id); perr != nil {
		c.respondError(ctx, perr)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func floatToMoneyPtr(f *float64) *decimal.Decimal {
	if f == nil {
		return nil
	}
	d := decimal.NewFromFloat(*f)
	return &d
}

// respondError maps a models.Error to an HTTP status + envelope.
func (c *ProductController) respondError(ctx *gin.Context, e *models.Error) {
	status := http.StatusUnprocessableEntity
	switch e.Code {
	case models.CodeRepositoryFailure:
		status = http.StatusInternalServerError
	case models.CodeProductNotFound:
		status = http.StatusNotFound
	}
	if status >= 500 {
		slog.ErrorContext(ctx.Request.Context(), "product.controller_error", "code", e.Code, "field", e.Field, "message", e.Message)
		writeServerError(ctx)
		return
	}
	writeError(ctx, status, e.Code)
}
