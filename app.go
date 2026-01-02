package main

import (
	"bus_history/internal/collector"
	"bus_history/internal/config"
	"bus_history/internal/model"
	"bus_history/internal/repository"
	"bus_history/internal/service"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context

	settings *config.AppSettings
	cfg      *config.Config

	db         *sql.DB
	busRepo    *repository.BusRepository
	configRepo *repository.ConfigRepository
	apiClient  *service.OpenAPIClient
	gbisClient *service.GBISClient
	busService *service.BusService
	collector  *collector.Collector

	mu sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load settings
	settings, err := config.LoadAppSettings()
	if err != nil {
		log.Printf("Failed to load settings: %v", err)
		return
	}
	a.settings = settings

	if settings.StoragePath != "" && settings.ServiceKey != "" {
		if err := a.initializeServices(); err != nil {
			log.Printf("Failed to initialize services: %v", err)
		}
	}
}

func (a *App) initializeServices() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Shutdown existing services if active
	if a.collector != nil {
		a.collector.Stop()
	}
	if a.db != nil {
		a.db.Close()
	}

	a.cfg = config.LoadFromSettings(a.settings)

	// Create storage dir if not exists
	if err := os.MkdirAll(a.settings.StoragePath, 0755); err != nil {
		return err
	}

	// Init DB
	db, err := sql.Open("sqlite3", a.cfg.Database.FilePath)
	if err != nil {
		return err
	}
	a.db = db

	// Run migrations
	a.runInitSchema()

	// Init Repos
	a.busRepo = repository.NewBusRepository(db)
	a.configRepo = repository.NewConfigRepository(db)

	// Init Clients (Passing the same service key to both)
	a.apiClient = service.NewOpenAPIClient(a.cfg.OpenAPI.BaseURL, a.cfg.OpenAPI.ServiceKey)
	a.gbisClient = service.NewGBISClient(a.cfg.OpenAPI.ServiceKey)

	incheonClient := service.NewIncheonClient(a.cfg.OpenAPI.ServiceKey)
	a.busService = service.NewBusService(a.gbisClient, incheonClient)

	// Init Collector
	a.collector = collector.NewCollector(
		a.configRepo,
		a.busRepo,
		a.apiClient,
		a.gbisClient,
		a.cfg.Collector.IntervalMs,
		a.settings.StartHour,
		a.settings.EndHour,
	)

	return nil
}

func (a *App) runInitSchema() {
	schema := `
	CREATE TABLE IF NOT EXISTS route_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		route_id TEXT NOT NULL,
		route_name TEXT NOT NULL,
		station_id TEXT NOT NULL,
		station_name TEXT NOT NULL,
		direction TEXT NOT NULL DEFAULT '',
		sta_order INTEGER NOT NULL DEFAULT 0,
		is_active BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS bus_arrivals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		route_config_id INTEGER NOT NULL,
		bus_number TEXT NOT NULL,
		arrival_time DATETIME NOT NULL,
		seats_before INTEGER,
		seats_after INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (route_config_id) REFERENCES route_configs(id)
	);
	`
	_, err := a.db.Exec(schema)
	if err != nil {
		log.Printf("Failed to init schema: %v", err)
	}
}

// --- Bindings for Settings ---

func (a *App) GetSettings() *config.AppSettings {
	return a.settings
}

func (a *App) UpdateSettings(storagePath, serviceKey string, startHour, endHour, intervalMs int) error {
	a.settings.StoragePath = storagePath
	a.settings.ServiceKey = serviceKey
	a.settings.StartHour = startHour
	a.settings.EndHour = endHour
	a.settings.IntervalMs = intervalMs

	if err := config.SaveAppSettings(a.settings); err != nil {
		return err
	}

	return a.initializeServices()
}

// --- Bindings for Collector Control ---

func (a *App) StartCollection() error {
	if a.collector == nil {
		return fmt.Errorf("app not initialized. Please check settings.")
	}
	return a.collector.Start(a.ctx)
}

func (a *App) StopCollection() {
	if a.collector != nil {
		a.collector.Stop()
	}
}

func (a *App) GetCollectionStatus() bool {
	if a.collector == nil {
		return false
	}
	return a.collector.IsRunning()
}

// --- Bindings for Data ---

func (a *App) SearchRoutes(keyword string) ([]model.RouteInfo, error) {
	if a.busService == nil {
		return nil, fmt.Errorf("system not initialized")
	}
	return a.busService.SearchRoutes(a.ctx, keyword)
}

func (a *App) GetRouteStations(routeID string, region string) ([]model.RouteStation, error) {
	if a.busService == nil {
		return nil, fmt.Errorf("system not initialized")
	}
	return a.busService.GetRouteStations(a.ctx, routeID, region)
}

func (a *App) SearchStations(keyword string) ([]model.StationInfo, error) {
	if a.busService == nil {
		return nil, fmt.Errorf("system not initialized")
	}
	return a.busService.SearchStations(a.ctx, keyword)
}

func (a *App) GetStationRoutes(stationID string, region string) ([]service.StationRouteInfo, error) {
	if a.busService == nil {
		return nil, fmt.Errorf("system not initialized")
	}
	return a.busService.GetStationRoutes(a.ctx, stationID, region)
}

func (a *App) GetConfigs() ([]*model.RouteConfig, error) {
	if a.configRepo == nil {
		return nil, fmt.Errorf("DB not initialized")
	}
	return a.configRepo.FindAll()
}

func (a *App) CreateConfig(cfg *model.RouteConfig) error {
	if a.configRepo == nil {
		return fmt.Errorf("DB not initialized")
	}

	// Ensure always active on registration
	cfg.IsActive = true

	err := a.configRepo.Create(cfg)
	if err != nil {
		return err
	}

	// Auto-start collector if not running
	if a.collector != nil {
		if !a.collector.IsRunning() {
			a.collector.Start(a.ctx)
		}
		a.collector.NotifySync()
	}
	return nil
}

func (a *App) DeleteConfig(id int64) error {
	if a.configRepo == nil {
		return fmt.Errorf("DB not initialized")
	}
	return a.configRepo.Delete(id)
}

func (a *App) ToggleConfig(id int64, active bool) error {
	if a.configRepo == nil {
		return fmt.Errorf("DB not initialized")
	}
	return a.configRepo.UpdateStatus(id, active)
}

func (a *App) GetArrivals(routeID, stationID, fromDate, toDate string, page, limit int) (map[string]interface{}, error) {
	if a.busRepo == nil {
		return nil, fmt.Errorf("DB not initialized")
	}

	filter := model.BusArrivalFilter{
		RouteID:   routeID,
		StationID: stationID,
		Page:      page,
		Limit:     limit,
	}

	loc, _ := time.LoadLocation("Asia/Seoul")
	if fromDate != "" {
		t, _ := time.ParseInLocation("2006-01-02", fromDate, loc)
		filter.FromDate = &t
	}
	if toDate != "" {
		t, _ := time.ParseInLocation("2006-01-02", toDate, loc)
		endOfDay := t.Add(24*time.Hour - time.Second)
		filter.ToDate = &endOfDay
	}

	arrivals, total, err := a.busRepo.FindByFilter(filter)
	if err != nil {
		return nil, err
	}

	// Ensure data is an empty array instead of null
	if arrivals == nil {
		arrivals = []*model.BusArrivalWithConfig{}
	}

	return map[string]interface{}{
		"data":  arrivals,
		"total": total,
		"page":  page,
		"limit": limit,
	}, nil
}

func (a *App) GetTrip(arrivalID int64) ([]*model.BusArrivalWithConfig, error) {
	if a.busRepo == nil {
		return nil, fmt.Errorf("DB not initialized")
	}
	return a.busRepo.GetTripByArrivalID(arrivalID)
}

// SelectFolder opens a native directory dialog and returns the selected path
func (a *App) SelectFolder() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "데이터 저장 폴더 선택",
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}
