// Component test: boots the full DI graph, applies migrations, and
// drives the HTTP surface with httptest. Skips cleanly if Postgres
// is unreachable.
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Touutae-labs/simple-gin/internal/configurations"
	"github.com/Touutae-labs/simple-gin/internal/di"
	"github.com/Touutae-labs/simple-gin/internal/repositories"
	"github.com/Touutae-labs/simple-gin/internal/server"
)

func init() {
	gin.SetMode(gin.TestMode)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}


// testDB returns a *gorm.DB to the test database, with the schema
// applied and the table truncated.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=simple_gin_test port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.New(postgres.Config{DriverName: "pgx", DSN: dsn}),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("skipping component test: cannot connect to Postgres: %v", err)
	}

	if err := db.AutoMigrate(repositories.AllModels()...); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	if err := db.Exec("TRUNCATE products").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}

	return db
}


func testApp(t *testing.T) *gin.Engine {
	t.Helper()
	testDB(t)
	cfg := loadConfigOrDefault(t)
	app, cleanup, err := di.Initialize(cfg, server.ServerTitle("simple-gin-test"), server.ServerVersion("test"))
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	t.Cleanup(cleanup)
	return app.Server.App
}


// do runs req through the engine in-process via httptest.NewRecorder
// + ServeHTTP. Returns the response with body still readable.
func do(t *testing.T, app *gin.Engine, req *http.Request) *http.Response {
	t.Helper()
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	return w.Result()
}


func TestHealth(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", resp.StatusCode)
	}
}


func TestCreateProduct_HappyPath(t *testing.T) {
	app := testApp(t)
	body := bytes.NewBufferString(`{"name":"Espresso","price":29.90}`)
	req := httptest.NewRequest(http.MethodPost, "/product", body)
	req.Header.Set("Content-Type", "application/json")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(b))
	}

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if ok, _ := out["successful"].(bool); !ok {
		t.Errorf("expected successful=true, got %v", out)
	}

	data, _ := out["data"].(map[string]any)
	if data == nil || data["data1"] == "" {
		t.Errorf("expected data.data1 to be the product id, got %v", out)
	}
}


func TestCreateProduct_InvalidPrice(t *testing.T) {
	app := testApp(t)
	body := bytes.NewBufferString(`{"name":"x","price":-1}`)
	req := httptest.NewRequest(http.MethodPost, "/product", body)
	req.Header.Set("Content-Type", "application/json")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d, want 422", resp.StatusCode)
	}

	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["error_code"] != "INVALID_PRICE" {
		t.Errorf("expected error_code=INVALID_PRICE, got %v", out)
	}
}


func TestCreateProduct_InvalidBody(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequest(http.MethodPost, "/product", bytes.NewBufferString(`{not json`))
	req.Header.Set("Content-Type", "application/json")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}


func TestPatchProduct_NotFound(t *testing.T) {
	app := testApp(t)
	body := bytes.NewBufferString(`{"price":12.50}`)
	req := httptest.NewRequest(http.MethodPatch, "/product/00000000-0000-0000-0000-000000000000", body)
	req.Header.Set("Content-Type", "application/json")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", resp.StatusCode)
	}
}


func TestPatchProduct_InvalidBody(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequest(http.MethodPatch, "/product/x", bytes.NewBufferString(`{not json`))
	req.Header.Set("Content-Type", "application/json")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}


func TestCreateThenPatch_FullFlow(t *testing.T) {
	app := testApp(t)

	createBody := bytes.NewBufferString(`{"name":"Mug","description":"ceramic","price":9.99}`)
	req := httptest.NewRequest(http.MethodPost, "/product", createBody)
	req.Header.Set("Content-Type", "application/json")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status=%d", resp.StatusCode)
	}

	var created map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&created)
	data, _ := created["data"].(map[string]any)
	id, _ := data["data1"].(string)
	if id == "" {
		t.Fatalf("no id in create response: %v", created)
	}


	patchBody := bytes.NewBufferString(`{"price":12.50}`)
	req = httptest.NewRequest(http.MethodPatch, "/product/"+id, patchBody)
	req.Header.Set("Content-Type", "application/json")
	resp = do(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch: status=%d", resp.StatusCode)
	}

	var patched map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&patched)
	if ok, _ := patched["successful"].(bool); !ok {
		t.Errorf("expected successful=true, got %v", patched)
	}
}


func loadConfigOrDefault(t *testing.T) configurations.Config {
	t.Helper()
	if path := os.Getenv("APP_CONFIG"); path != "" {
		if cfg, err := readYAML(path); err == nil {
			return cfg
		}
	}

	if _, err := os.Stat("config.yml"); err == nil {
		if cfg, err := readYAML("config.yml"); err == nil {
			return cfg
		}
	}

	return configurations.Config{
		Server:   configurations.ServerConfig{Port: "0", MaxPayloadSizeKB: 4096, TimeoutSeconds: 30, BaseURL: "http://localhost:8080"},
		Database: configurations.DatabaseConfig{Host: "localhost", Port: 5432, User: "postgres", Password: "postgres", DBName: "simple_gin_test", SSLMode: "disable", AutoMigrate: true},
	}
}


func readYAML(path string) (configurations.Config, error) {
	var cfg configurations.Config
	k := koanf.New(".")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return cfg, err
	}

	if err := k.Unmarshal("", &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
