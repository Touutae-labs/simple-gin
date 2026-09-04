package testhelpers_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/Touutae-labs/simple-gin/internal/models"
	"github.com/Touutae-labs/simple-gin/internal/testhelpers"
)

func TestProductMemory_Patch_ExplicitClearDescription(t *testing.T) {
	repo := testhelpers.NewProductMemory()
	desc := "to be cleared"
	id, err := repo.Create(context.Background(), models.CreateInput{
		Name: "X", Description: &desc, Price: decimal.RequireFromString("1"),
	})
	require.NoError(t, err)

	empty := ""
	_, err = repo.Patch(context.Background(), id, models.PatchInput{Description: &empty})
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.Nil(t, got.Description, "empty string in PATCH should clear the column to NULL")
}
