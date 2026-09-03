package di

import (
	"gorm.io/gorm"

	"github.com/Touutae-labs/simple-gin/internal/server"
)

// Application is the fully-wired process. cmd/server reads Server
// out of it; the seed/migrate commands reach DB directly.
//
// Defined in its own file (no build tag) so the wire_gen.go
// generated code can reference it under the !wireinject constraint.
type Application struct {
	Server *server.Server
	DB     *gorm.DB
}
