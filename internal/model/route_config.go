package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// RouteConfig represents a monitoring configuration for a bus route at a station
type RouteConfig struct {
	ID          int64     `json:"id" db:"id"`
	RouteID     string    `json:"route_id" db:"route_id"`
	RouteName   string    `json:"route_name" db:"route_name"`
	StationID   string    `json:"station_id" db:"station_id"`
	StationName string    `json:"station_name" db:"station_name"`
	Direction   string    `json:"direction" db:"direction"`
	StaOrder    int       `json:"sta_order" db:"sta_order"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateRouteConfigRequest represents the request to create a new route config
type CreateRouteConfigRequest struct {
	RouteID     int    `json:"-"` // Use custom unmarshaler
	RouteName   string `json:"route_name" binding:"required"`
	StationID   int    `json:"-"` // Use custom unmarshaler
	StationName string `json:"station_name" binding:"required"`
	Direction   string `json:"direction"`
	StaOrder    int    `json:"sta_order"`
}

// UnmarshalJSON custom unmarshaler to handle route_id/station_id as both string and int
func (r *CreateRouteConfigRequest) UnmarshalJSON(data []byte) error {
	type Alias struct {
		RouteID     interface{} `json:"route_id"`
		RouteName   string      `json:"route_name"`
		StationID   interface{} `json:"station_id"`
		StationName string      `json:"station_name"`
		Direction   string      `json:"direction"`
		StaOrder    int         `json:"sta_order"`
	}

	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	r.RouteName = aux.RouteName
	r.StationName = aux.StationName
	r.Direction = aux.Direction
	r.StaOrder = aux.StaOrder

	// Parse RouteID (can be string or number)
	switch v := aux.RouteID.(type) {
	case string:
		if val, err := strconv.Atoi(v); err == nil {
			r.RouteID = val
		} else {
			return fmt.Errorf("invalid route_id: %s", v)
		}
	case float64:
		r.RouteID = int(v)
	case int:
		r.RouteID = v
	default:
		return fmt.Errorf("route_id must be a number or string")
	}

	// Parse StationID (can be string or number)
	switch v := aux.StationID.(type) {
	case string:
		if val, err := strconv.Atoi(v); err == nil {
			r.StationID = val
		} else {
			return fmt.Errorf("invalid station_id: %s", v)
		}
	case float64:
		r.StationID = int(v)
	case int:
		r.StationID = v
	default:
		return fmt.Errorf("station_id must be a number or string")
	}

	return nil
}

// UpdateRouteConfigRequest represents the request to update a route config
type UpdateRouteConfigRequest struct {
	StationName string `json:"station_name,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}
