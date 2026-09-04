package product

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Touutae-labs/simple-gin/internal/models"
)

// Service is the orchestrator. It runs the module's rules and
// delegates persistence to the repository port.
type Service interface {
	Create(ctx context.Context, in models.CreateInput) (models.Result, *models.Error)
	Get(ctx context.Context, id string) (*models.Product, *models.Error)
	List(ctx context.Context, f *models.ListFilter) (*models.ListPage, *models.Error)
	Patch(ctx context.Context, id string, in models.PatchInput) (models.Result, *models.Error)
	Delete(ctx context.Context, id string) *models.Error
}


type serviceImpl struct {
	mod  Module
	repo Repository
}


// NewService composes the module and the repository. The returned
// concrete type is unexported so the controller depends on the interface.
func NewService(mod Module, repo Repository) Service {
	return &serviceImpl{mod: mod, repo: repo}
}


func (s *serviceImpl) Create(ctx context.Context, in models.CreateInput) (models.Result, *models.Error) {
	if err := s.mod.ValidateCreate(in.Name, in.Description, in.Price); err != nil {
		return models.Result{}, err
	}

	id, err := s.repo.Create(ctx, in)
	if err != nil {
		return models.Result{}, repoErr(ctx, err, "create", "")
	}

	return models.Result{ProductID: id, Name: in.Name}, nil
}


func (s *serviceImpl) Get(ctx context.Context, id string) (*models.Product, *models.Error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, repoErr(ctx, err, "get", id)
	}

	return p, nil
}


func (s *serviceImpl) List(ctx context.Context, f *models.ListFilter) (*models.ListPage, *models.Error) {
	if f == nil {
		f = &models.ListFilter{}
	}

	if err := s.mod.ValidateListFilter(f); err != nil {
		return nil, err
	}

	page, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, repoErr(ctx, err, "list", "")
	}

	return page, nil
}


func (s *serviceImpl) Patch(ctx context.Context, id string, in models.PatchInput) (models.Result, *models.Error) {
	if err := s.mod.ValidatePatch(in.Name, in.Description, in.Price); err != nil {
		return models.Result{}, err
	}

	updated, err := s.repo.Patch(ctx, id, in)
	if err != nil {
		return models.Result{}, repoErr(ctx, err, "patch", id)
	}

	return models.Result{ProductID: updated.ID, Name: updated.Name}, nil
}


func (s *serviceImpl) Delete(ctx context.Context, id string) *models.Error {
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return repoErr(ctx, err, "delete", id)
	}

	return nil
}


// repoErr maps a Repository error to the wire error the controller
// expects. ErrNotFound → PRODUCT_NOT_FOUND; anything else → log a
// structured event and return REPOSITORY_FAILURE. id is optional —
// pass "" for operations that don't have one (Create, List).
func repoErr(ctx context.Context, err error, op, id string) *models.Error {
	if errors.Is(err, ErrNotFound) {
		return &models.Error{
			Code:    models.CodeProductNotFound,
			Field:   "id",
			Message: fmt.Sprintf("product %s not found", id),
		}
	}

	attrs := []slog.Attr{slog.String("err", err.Error())}
	if id != "" {
		attrs = append(attrs, slog.String("id", id))
	}

	slog.LogAttrs(ctx, slog.LevelError, "product."+op+".repo_error", attrs...)
	return &models.Error{
		Code:    models.CodeRepositoryFailure,
		Message: "failed to " + op + " product",
	}
}
