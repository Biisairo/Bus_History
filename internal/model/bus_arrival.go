package model

import "time"

// BusArrival represents a bus arrival record
type BusArrival struct {
	ID            int64     `json:"id" db:"id"`
	RouteConfigID int64     `json:"route_config_id" db:"route_config_id"`
	BusNumber     string    `json:"bus_number" db:"bus_number"`
	ArrivalTime   time.Time `json:"arrival_time" db:"arrival_time"`
	SeatsBefore   *int      `json:"seats_before" db:"seats_before"`
	SeatsAfter    *int      `json:"seats_after" db:"seats_after"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// BusArrivalWithConfig represents a bus arrival with route config information
type BusArrivalWithConfig struct {
	BusArrival
	RouteID     string `json:"route_id" db:"route_id"`
	RouteName   string `json:"route_name" db:"route_name"`
	StationID   string `json:"station_id" db:"station_id"`
	StationName string `json:"station_name" db:"station_name"`
	StaOrder    int    `json:"sta_order" db:"sta_order"`
}

// BusArrivalFilter represents filters for querying bus arrivals
type BusArrivalFilter struct {
	RouteID   string
	StationID string
	FromDate  *time.Time
	ToDate    *time.Time
	Page      int
	Limit     int
}

// BusArrivalStats represents statistics for bus arrivals
type BusArrivalStats struct {
	RouteID       string   `json:"route_id"`
	StationName   string   `json:"station_name"`
	PeriodFrom    string   `json:"period_from"`
	PeriodTo      string   `json:"period_to"`
	TotalArrivals int      `json:"total_arrivals"`
	AvgBefore     float64  `json:"avg_seats_before"`
	AvgAfter      float64  `json:"avg_seats_after"`
	AvgBoarding   float64  `json:"avg_boarding"`
	BusiestHours  []string `json:"busiest_hours"`
}

// APIResponse is a generic API response wrapper
type APIResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total,omitempty"`
	Page    int         `json:"page,omitempty"`
	Limit   int         `json:"limit,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}
