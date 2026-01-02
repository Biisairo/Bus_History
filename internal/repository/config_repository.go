package repository

import (
	"bus_history/internal/model"
	"database/sql"
	"fmt"
)

// ConfigRepository handles route config database operations
type ConfigRepository struct {
	db *sql.DB
}

// NewConfigRepository creates a new config repository
func NewConfigRepository(db *sql.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

// FindAll retrieves all route configs
func (r *ConfigRepository) FindAll() ([]*model.RouteConfig, error) {
	query := `SELECT id, route_id, route_name, station_id, station_name, direction, sta_order, is_active, created_at, updated_at 
			  FROM route_configs ORDER BY route_name ASC, sta_order ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query route configs: %w", err)
	}
	defer rows.Close()

	var configs []*model.RouteConfig
	for rows.Next() {
		var cfg model.RouteConfig
		if err := rows.Scan(&cfg.ID, &cfg.RouteID, &cfg.RouteName, &cfg.StationID, &cfg.StationName, &cfg.Direction, &cfg.StaOrder,
			&cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan route config: %w", err)
		}
		configs = append(configs, &cfg)
	}

	return configs, rows.Err()
}

// FindByID retrieves a route config by ID
func (r *ConfigRepository) FindByID(id int64) (*model.RouteConfig, error) {
	query := `SELECT id, route_id, route_name, station_id, station_name, direction, sta_order, is_active, created_at, updated_at 
			  FROM route_configs WHERE id = ?`

	var cfg model.RouteConfig
	err := r.db.QueryRow(query, id).Scan(&cfg.ID, &cfg.RouteID, &cfg.RouteName, &cfg.StationID,
		&cfg.StationName, &cfg.Direction, &cfg.StaOrder, &cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query route config: %w", err)
	}

	return &cfg, nil
}

// FindActive retrieves all active route configs
func (r *ConfigRepository) FindActive() ([]*model.RouteConfig, error) {
	query := `SELECT id, route_id, route_name, station_id, station_name, direction, sta_order, is_active, created_at, updated_at 
			  FROM route_configs WHERE is_active = TRUE ORDER BY route_name ASC, sta_order ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active route configs: %w", err)
	}
	defer rows.Close()

	var configs []*model.RouteConfig
	for rows.Next() {
		var cfg model.RouteConfig
		if err := rows.Scan(&cfg.ID, &cfg.RouteID, &cfg.RouteName, &cfg.StationID, &cfg.StationName, &cfg.Direction, &cfg.StaOrder,
			&cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan route config: %w", err)
		}
		configs = append(configs, &cfg)
	}

	return configs, rows.Err()
}

// Create creates a new route config
func (r *ConfigRepository) Create(cfg *model.RouteConfig) error {
	query := `INSERT INTO route_configs (route_id, route_name, station_id, station_name, direction, sta_order, is_active) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query, cfg.RouteID, cfg.RouteName, cfg.StationID, cfg.StationName, cfg.Direction, cfg.StaOrder, cfg.IsActive)
	if err != nil {
		return fmt.Errorf("failed to create route config: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	cfg.ID = id
	return nil
}

// Update updates an existing route config
func (r *ConfigRepository) Update(id int64, stationName *string, isActive *bool) error {
	query := "UPDATE route_configs SET"
	args := []interface{}{}
	updates := []string{}

	if stationName != nil {
		updates = append(updates, " station_name = ?")
		args = append(args, *stationName)
	}
	if isActive != nil {
		updates = append(updates, " is_active = ?")
		args = append(args, *isActive)
	}

	if len(updates) == 0 {
		return nil
	}

	query += updates[0]
	for i := 1; i < len(updates); i++ {
		query += "," + updates[i]
	}
	query += " WHERE id = ?"
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update route config: %w", err)
	}

	return nil
}

// Delete deletes a route config by ID
func (r *ConfigRepository) Delete(id int64) error {
	query := "DELETE FROM route_configs WHERE id = ?"
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete route config: %w", err)
	}
	return nil
}

// UpdateStatus updates the is_active status of a route config
func (r *ConfigRepository) UpdateStatus(id int64, isActive bool) error {
	query := "UPDATE route_configs SET is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := r.db.Exec(query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update route config status: %w", err)
	}
	return nil
}
