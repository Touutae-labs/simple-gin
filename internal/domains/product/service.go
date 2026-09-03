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
	Patch(ctx context.Context, id string, in models.PatchInput) (models.Result, *models.Error)
}

type serviceImpl struct {
	mod  Module
	repo Repository
}

// NewService composes the module and the repository. The returned
// concrete type is unexported so the controller depends on the
// interface.
func NewService(mod Module, repo Repository) Service {
	return &serviceImpl{mod: mod, repo: repo}
}

func (s *serviceImpl) Create(ctx context.Context, in models.CreateInput) (models.Result, *models.Error) {
	if err := s.mod.ValidateCreate(in.Name, in.Price); err != nil {
		return models.Result{}, err
	}
	id, err := s.repo.Create(ctx, in)
	if err != nil {
		slog.ErrorContext(ctx, "product.create.repo_error", "err", err.Error())
		return models.Result{}, &models.Error{
			Code:    models.CodeRepositoryFailure,
			Message: "failed to create product",
		}
	}
	return models.Result{ProductID: id, Name: in.Name}, nil
}

func (s *serviceImpl) Patch(ctx context.Context, id string, in models.PatchInput) (models.Result, *models.Error) {
	if err := s.mod.ValidatePatch(in.Name, in.Price); err != nil {
		return models.Result{}, err
	}
	updated, err := s.repo.Patch(ctx, id, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Result{}, &models.Error{
				Code:    models.CodeProductNotFound,
				Field:   "id",
				Message: fmt.Sprintf("product %s not found", id),
			}
		}
		slog.ErrorContext(ctx, "product.patch.repo_error", "id", id, "err", err.Error())
		return models.Result{}, &models.Error{
			Code:    models.CodeRepositoryFailure,
			Message: "failed to update product",
		}
	}
	return models.Result{ProductID: updated.ID, Name: updated.Name}, nil
}
