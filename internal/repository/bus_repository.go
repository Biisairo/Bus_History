package repository

import (
	"bus_history/internal/model"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// BusRepository handles bus arrival database operations
type BusRepository struct {
	db *sql.DB
}

// NewBusRepository creates a new bus repository
func NewBusRepository(db *sql.DB) *BusRepository {
	return &BusRepository{db: db}
}

// Create creates a new bus arrival record
func (r *BusRepository) Create(arrival *model.BusArrival) error {
	query := `INSERT INTO bus_arrivals (route_config_id, bus_number, arrival_time, seats_before, seats_after) 
			  VALUES (?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query, arrival.RouteConfigID, arrival.BusNumber,
		arrival.ArrivalTime, arrival.SeatsBefore, arrival.SeatsAfter)
	if err != nil {
		return fmt.Errorf("failed to create bus arrival: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	arrival.ID = id
	return nil
}

// UpdateSeatsAfter updates the seats_after field for a bus arrival
func (r *BusRepository) UpdateSeatsAfter(id int64, seatsAfter int) error {
	query := "UPDATE bus_arrivals SET seats_after = ? WHERE id = ?"
	_, err := r.db.Exec(query, seatsAfter, id)
	if err != nil {
		return fmt.Errorf("failed to update seats after: %w", err)
	}
	return nil
}

// FindByID retrieves a bus arrival by ID with config info
func (r *BusRepository) FindByID(id int64) (*model.BusArrivalWithConfig, error) {
	query := `SELECT ba.id, ba.route_config_id, ba.bus_number, ba.arrival_time, 
					 ba.seats_before, ba.seats_after, ba.created_at,
					 rc.route_id, rc.route_name, rc.station_id, rc.station_name, rc.sta_order
			  FROM bus_arrivals ba
			  JOIN route_configs rc ON ba.route_config_id = rc.id
			  WHERE ba.id = ?`

	var arrival model.BusArrivalWithConfig
	err := r.db.QueryRow(query, id).Scan(
		&arrival.ID, &arrival.RouteConfigID, &arrival.BusNumber, &arrival.ArrivalTime,
		&arrival.SeatsBefore, &arrival.SeatsAfter, &arrival.CreatedAt,
		&arrival.RouteID, &arrival.RouteName, &arrival.StationID, &arrival.StationName, &arrival.StaOrder,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query bus arrival: %w", err)
	}

	return &arrival, nil
}

// FindByFilter retrieves bus arrivals with filters
func (r *BusRepository) FindByFilter(filter model.BusArrivalFilter) ([]*model.BusArrivalWithConfig, int64, error) {
	// Build query
	baseQuery := `FROM bus_arrivals ba JOIN route_configs rc ON ba.route_config_id = rc.id`
	where := []string{}
	args := []interface{}{}

	if filter.RouteID != "" {
		where = append(where, "rc.route_id = ?")
		args = append(args, filter.RouteID)
	}
	if filter.StationID != "" {
		where = append(where, "rc.station_id = ?")
		args = append(args, filter.StationID)
	}
	if filter.FromDate != nil {
		where = append(where, "ba.arrival_time >= ?")
		args = append(args, filter.FromDate)
	}
	if filter.ToDate != nil {
		where = append(where, "ba.arrival_time <= ?")
		args = append(args, filter.ToDate)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery + whereClause
	var total int64
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count bus arrivals: %w", err)
	}

	// Get paginated results
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	selectQuery := `SELECT ba.id, ba.route_config_id, ba.bus_number, ba.arrival_time, 
						   ba.seats_before, ba.seats_after, ba.created_at,
						   rc.route_id, rc.route_name, rc.station_id, rc.station_name, rc.sta_order ` +
		baseQuery + whereClause + " ORDER BY ba.arrival_time DESC LIMIT ? OFFSET ?"

	args = append(args, filter.Limit, offset)
	rows, err := r.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query bus arrivals: %w", err)
	}
	defer rows.Close()

	var arrivals []*model.BusArrivalWithConfig
	for rows.Next() {
		var arrival model.BusArrivalWithConfig
		if err := rows.Scan(
			&arrival.ID, &arrival.RouteConfigID, &arrival.BusNumber, &arrival.ArrivalTime,
			&arrival.SeatsBefore, &arrival.SeatsAfter, &arrival.CreatedAt,
			&arrival.RouteID, &arrival.RouteName, &arrival.StationID, &arrival.StationName, &arrival.StaOrder,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan bus arrival: %w", err)
		}
		arrivals = append(arrivals, &arrival)
	}

	return arrivals, total, rows.Err()
}

// GetStatistics retrieves statistics for a route/station combination
func (r *BusRepository) GetStatistics(routeID, stationID string, fromDate, toDate *time.Time) (*model.BusArrivalStats, error) {
	query := `SELECT 
				rc.route_id,
				rc.station_name,
				COUNT(*) as total_arrivals,
				AVG(ba.seats_before) as avg_before,
				AVG(ba.seats_after) as avg_after,
				AVG(ba.seats_before - ba.seats_after) as avg_boarding
			  FROM bus_arrivals ba
			  JOIN route_configs rc ON ba.route_config_id = rc.id
			  WHERE rc.route_id = ? AND rc.station_id = ?`

	args := []interface{}{routeID, stationID}

	if fromDate != nil {
		query += " AND ba.arrival_time >= ?"
		args = append(args, fromDate)
	}
	if toDate != nil {
		query += " AND ba.arrival_time <= ?"
		args = append(args, toDate)
	}

	query += " GROUP BY rc.route_id, rc.station_name"

	var stats model.BusArrivalStats
	var avgBefore, avgAfter, avgBoarding sql.NullFloat64

	err := r.db.QueryRow(query, args...).Scan(
		&stats.RouteID, &stats.StationName, &stats.TotalArrivals,
		&avgBefore, &avgAfter, &avgBoarding,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	if avgBefore.Valid {
		stats.AvgBefore = avgBefore.Float64
	}
	if avgAfter.Valid {
		stats.AvgAfter = avgAfter.Float64
	}
	if avgBoarding.Valid {
		stats.AvgBoarding = avgBoarding.Float64
	}

	// Get busiest hours
	hourQuery := `SELECT HOUR(ba.arrival_time) as hour, COUNT(*) as count
				  FROM bus_arrivals ba
				  JOIN route_configs rc ON ba.route_config_id = rc.id
				  WHERE rc.route_id = ? AND rc.station_id = ?`

	hourArgs := []interface{}{routeID, stationID}
	if fromDate != nil {
		hourQuery += " AND ba.arrival_time >= ?"
		hourArgs = append(hourArgs, fromDate)
	}
	if toDate != nil {
		hourQuery += " AND ba.arrival_time <= ?"
		hourArgs = append(hourArgs, toDate)
	}

	hourQuery += " GROUP BY HOUR(ba.arrival_time) ORDER BY count DESC LIMIT 3"

	rows, err := r.db.Query(hourQuery, hourArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get busiest hours: %w", err)
	}
	defer rows.Close()

	stats.BusiestHours = []string{}
	for rows.Next() {
		var hour, count int
		if err := rows.Scan(&hour, &count); err != nil {
			return nil, fmt.Errorf("failed to scan hour: %w", err)
		}
		stats.BusiestHours = append(stats.BusiestHours,
			fmt.Sprintf("%02d:00-%02d:00", hour, hour+1))
	}

	// Set period
	if fromDate != nil {
		stats.PeriodFrom = fromDate.Format("2006-01-02")
	}
	if toDate != nil {
		stats.PeriodTo = toDate.Format("2006-01-02")
	}

	return &stats, nil
}

// GetTripByArrivalID identifies and returns the full trip sequence for a given arrival record
func (r *BusRepository) GetTripByArrivalID(id int64) ([]*model.BusArrivalWithConfig, error) {
	// 1. Get the target arrival to know busNumber and routeID
	target, err := r.FindByID(id)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, nil
	}

	// 2. Fetch all arrivals for this bus and route within a 12-hour window (to avoid loading too much history)
	// We use the target's arrival time as the center.
	startTime := target.ArrivalTime.Add(-6 * time.Hour)
	endTime := target.ArrivalTime.Add(6 * time.Hour)

	query := `SELECT ba.id, ba.route_config_id, ba.bus_number, ba.arrival_time, 
					 ba.seats_before, ba.seats_after, ba.created_at,
					 rc.route_id, rc.route_name, rc.station_id, rc.station_name, rc.sta_order
			  FROM bus_arrivals ba
			  JOIN route_configs rc ON ba.route_config_id = rc.id
			  WHERE ba.bus_number = ? AND rc.route_id = ?
			  AND ba.arrival_time BETWEEN ? AND ?
			  ORDER BY ba.arrival_time ASC`

	rows, err := r.db.Query(query, target.BusNumber, target.RouteID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query trip: %w", err)
	}
	defer rows.Close()

	var allArrivals []*model.BusArrivalWithConfig
	targetIndex := -1

	for rows.Next() {
		var a model.BusArrivalWithConfig
		err := rows.Scan(
			&a.ID, &a.RouteConfigID, &a.BusNumber, &a.ArrivalTime,
			&a.SeatsBefore, &a.SeatsAfter, &a.CreatedAt,
			&a.RouteID, &a.RouteName, &a.StationID, &a.StationName, &a.StaOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trip arrival: %w", err)
		}
		if a.ID == id {
			targetIndex = len(allArrivals)
		}
		allArrivals = append(allArrivals, &a)
	}

	if targetIndex == -1 {
		return nil, nil
	}

	// 3. Find the contiguous trip segment
	// Go backwards from targetIndex
	startIdx := targetIndex
	for i := targetIndex - 1; i >= 0; i-- {
		// If the previous station order is less than current, it's the same trip
		// Note: We might miss some gap if the bus skipped a monitored station,
		// but as long as it's increasing, we assume it's the same trip.
		if allArrivals[i].StaOrder < allArrivals[i+1].StaOrder {
			startIdx = i
		} else {
			break
		}
	}

	// Go forwards from targetIndex
	endIdx := targetIndex
	for i := targetIndex + 1; i < len(allArrivals); i++ {
		if allArrivals[i].StaOrder > allArrivals[i-1].StaOrder {
			endIdx = i
		} else {
			break
		}
	}

	return allArrivals[startIdx : endIdx+1], nil
}
