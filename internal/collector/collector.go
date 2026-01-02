package collector

import (
	"bus_history/internal/model"
	"bus_history/internal/repository"
	"bus_history/internal/service"
	"context"
	"log"
	"sync"
	"time"
)

// BusState tracks the state of a bus approaching/at a station
type BusState struct {
	PlateNo     string
	FirstSeenAt time.Time
	LastSeenAt  time.Time
	SeatsBefore int  // Seats when bus was approaching
	LocationNo  int  // Location when first seen
	Recorded    bool // Whether we've recorded this arrival
	// For pending seats_after retry
	PendingArrivalID int64     // DB ID if saved without seats_after
	PassedAt         time.Time // When bus passed the station
	RetryCount       int       // Number of retry attempts
}

// configCollector manages collection for a single config
type configCollector struct {
	cfg      *model.RouteConfig
	stopChan chan struct{}
}

// Collector manages bus data collection
type Collector struct {
	configRepo *repository.ConfigRepository
	busRepo    *repository.BusRepository
	apiClient  *service.OpenAPIClient
	gbisClient *service.GBISClient
	intervalMs int

	// Track running collectors per config ID
	mu         sync.RWMutex
	collectors map[int64]*configCollector
	mainCtx    context.Context
	mainCancel context.CancelFunc
	wg         sync.WaitGroup
	startHour  int
	endHour    int
}

// IsRunning returns true if the collector is started
func (c *Collector) IsRunning() bool {
	return c.mainCancel != nil
}

// NewCollector creates a new collector
func NewCollector(
	configRepo *repository.ConfigRepository,
	busRepo *repository.BusRepository,
	apiClient *service.OpenAPIClient,
	gbisClient *service.GBISClient,
	intervalMs int,
	startHour int,
	endHour int,
) *Collector {
	return &Collector{
		configRepo: configRepo,
		busRepo:    busRepo,
		apiClient:  apiClient,
		gbisClient: gbisClient,
		intervalMs: intervalMs,
		collectors: make(map[int64]*configCollector),
		startHour:  startHour,
		endHour:    endHour,
	}
}

// Start begins the data collection process
func (c *Collector) Start(ctx context.Context) error {
	log.Println("Starting data collector...")

	c.mainCtx, c.mainCancel = context.WithCancel(ctx)

	// Initial load
	c.syncConfigs()

	// Periodically reload configs (every 30 seconds for faster response)
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-c.mainCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				c.syncConfigs()
			}
		}
	}()

	return nil
}

// Stop stops all collectors
func (c *Collector) Stop() {
	if c.mainCancel == nil {
		return
	}
	log.Println("Stopping data collector...")
	c.mainCancel()

	c.mu.Lock()
	for id, cc := range c.collectors {
		close(cc.stopChan)
		delete(c.collectors, id)
	}
	c.mu.Unlock()

	c.wg.Wait()
	c.mainCancel = nil
	c.mainCtx = nil
	log.Println("Data collector stopped")
}

// NotifySync triggers an immediate sync of configurations
func (c *Collector) NotifySync() {
	go c.syncConfigs()
}

// syncConfigs synchronizes running collectors with database configs
func (c *Collector) syncConfigs() {
	configs, err := c.configRepo.FindActive()
	if err != nil {
		log.Printf("[Collector] Error loading configs: %v", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Create a set of active config IDs
	activeIDs := make(map[int64]bool)
	for _, cfg := range configs {
		activeIDs[cfg.ID] = true
	}

	// Stop collectors for deleted/inactive configs
	for id, cc := range c.collectors {
		if !activeIDs[id] {
			log.Printf("[Collector] Stopping collector for deleted/inactive config %d (%s)", id, cc.cfg.StationName)
			close(cc.stopChan)
			delete(c.collectors, id)
		}
	}

	// Start collectors for new configs
	for _, cfg := range configs {
		if _, exists := c.collectors[cfg.ID]; !exists {
			log.Printf("[Collector] Starting new collector for config %d: route=%s (%s), station=%s (%s)",
				cfg.ID, cfg.RouteID, cfg.RouteName, cfg.StationID, cfg.StationName)

			cc := &configCollector{
				cfg:      cfg,
				stopChan: make(chan struct{}),
			}
			c.collectors[cfg.ID] = cc

			c.wg.Add(1)
			go c.collectForConfig(cc)
		}
	}

	log.Printf("[Collector] Synced: %d active collectors", len(c.collectors))
}

// collectForConfig collects data for a single route configuration
func (c *Collector) collectForConfig(cc *configCollector) {
	defer c.wg.Done()

	cfg := cc.cfg
	log.Printf("[Collector] Collection started for route %s (%s) at station %s (%s)",
		cfg.RouteID, cfg.RouteName, cfg.StationID, cfg.StationName)

	ticker := time.NewTicker(time.Duration(c.intervalMs) * time.Millisecond)
	defer ticker.Stop()

	// Track buses approaching/at this station
	busStates := make(map[string]*BusState)

	for {
		select {
		case <-c.mainCtx.Done():
			return
		case <-cc.stopChan:
			log.Printf("[Collector] Collection stopped for route %s at station %s",
				cfg.RouteID, cfg.StationName)
			return
		case <-ticker.C:
			// Check time window
			if c.isWithinTimeWindow() {
				c.collectData(cfg, busStates)
			} else {
				log.Printf("[Collector] Outside time window (%d-%d), skipping collection for %s",
					c.startHour, c.endHour, cfg.StationName)
			}
		}
	}
}

// collectData performs a single data collection cycle
func (c *Collector) collectData(cfg *model.RouteConfig, busStates map[string]*BusState) {
	log.Printf("[Collector] === Collecting data for route %s (%s) at station %s (%s) ===",
		cfg.RouteID, cfg.RouteName, cfg.StationID, cfg.StationName)

	// Get bus arrival information from API
	arrivals, err := c.apiClient.GetRouteArrivalList(cfg.RouteID, cfg.StationID)
	if err != nil {
		log.Printf("[Collector] Error fetching data for route %s at station %s: %v",
			cfg.RouteID, cfg.StationID, err)
		return
	}

	log.Printf("[Collector] API returned %d arrivals, currently tracking %d buses",
		len(arrivals), len(busStates))

	now := time.Now()
	currentBuses := make(map[string]bool)

	// Process current API results
	for _, arrival := range arrivals {
		if arrival.PlateNo == "" {
			continue
		}

		currentBuses[arrival.PlateNo] = true

		state, exists := busStates[arrival.PlateNo]

		if !exists {
			// New bus detected - start tracking
			busStates[arrival.PlateNo] = &BusState{
				PlateNo:     arrival.PlateNo,
				FirstSeenAt: now,
				LastSeenAt:  now,
				SeatsBefore: arrival.RemainSeatCnt,
				LocationNo:  arrival.LocationNo1,
				Recorded:    false,
			}
			log.Printf("[Tracking] New bus %s approaching station %s, location=%d stops away, seats=%d",
				arrival.PlateNo, cfg.StationName, arrival.LocationNo1, arrival.RemainSeatCnt)
		} else {
			// Update existing bus state
			state.LastSeenAt = now
			// Update seats before if bus is getting closer
			if arrival.LocationNo1 < state.LocationNo {
				state.SeatsBefore = arrival.RemainSeatCnt
				state.LocationNo = arrival.LocationNo1
				log.Printf("[Tracking] Bus %s getting closer: location=%d, seats=%d",
					arrival.PlateNo, arrival.LocationNo1, arrival.RemainSeatCnt)
			}
		}
	}

	// Check for buses that have passed the station (no longer in API results)
	for plateNo, state := range busStates {
		if !currentBuses[plateNo] {
			// Bus is no longer in API results - it has passed the station
			if !state.Recorded {
				if state.PassedAt.IsZero() {
					state.PassedAt = now
				}

				// Try to get seats after from bus location API
				seatsAfter := c.getSeatsAfterFromBusLocation(cfg.RouteID, plateNo)

				if seatsAfter != nil {
					// Got valid seat data - save the record
					busArrival := &model.BusArrival{
						RouteConfigID: cfg.ID,
						BusNumber:     plateNo,
						ArrivalTime:   state.LastSeenAt,
						SeatsBefore:   &state.SeatsBefore,
						SeatsAfter:    seatsAfter,
					}

					if err := c.busRepo.Create(busArrival); err != nil {
						log.Printf("[Collector] ❌ Error saving bus arrival: %v", err)
					} else {
						passengersBoarded := state.SeatsBefore - *seatsAfter
						log.Printf("[Collector] ✅ Recorded arrival: route=%s, station=%s, bus=%s, seats_before=%d, seats_after=%d, passengers=%d",
							cfg.RouteName, cfg.StationName, plateNo, state.SeatsBefore, *seatsAfter, passengersBoarded)
						state.Recorded = true
					}
				} else {
					// No valid seat data yet - retry
					state.RetryCount++
					timeSincePassed := now.Sub(state.PassedAt)

					// Retry for up to 2 minutes (bus should reach next station by then)
					if timeSincePassed < 2*time.Minute {
						log.Printf("[Collector] ⏳ Waiting for valid seat data for bus %s (retry %d, elapsed %s)",
							plateNo, state.RetryCount, timeSincePassed.Round(time.Second))
					} else {
						// Timeout - save without seats_after
						log.Printf("[Collector] ⚠️ Timeout waiting for seat data for bus %s, saving without seats_after", plateNo)

						busArrival := &model.BusArrival{
							RouteConfigID: cfg.ID,
							BusNumber:     plateNo,
							ArrivalTime:   state.LastSeenAt,
							SeatsBefore:   &state.SeatsBefore,
							SeatsAfter:    nil,
						}

						if err := c.busRepo.Create(busArrival); err != nil {
							log.Printf("[Collector] ❌ Error saving bus arrival: %v", err)
						} else {
							log.Printf("[Collector] ✅ Recorded arrival (no seats_after): route=%s, station=%s, bus=%s, seats_before=%d",
								cfg.RouteName, cfg.StationName, plateNo, state.SeatsBefore)
							state.Recorded = true
						}
					}
				}
			}

			// Remove bus from tracking after 10 minutes
			if now.Sub(state.LastSeenAt) > 10*time.Minute {
				delete(busStates, plateNo)
				log.Printf("[Cleanup] Removed bus %s from tracking", plateNo)
			}
		}
	}

	// Clean up very old entries
	for plateNo, state := range busStates {
		if now.Sub(state.FirstSeenAt) > 1*time.Hour {
			delete(busStates, plateNo)
		}
	}
}

// getSeatsAfterFromBusLocation queries the bus location API to get current seat count
func (c *Collector) getSeatsAfterFromBusLocation(routeID, plateNo string) *int {
	locations, err := c.gbisClient.GetBusLocations(routeID)
	if err != nil {
		log.Printf("[Collector] Error getting bus locations: %v", err)
		return nil
	}

	for _, loc := range locations {
		if loc.PlateNo == plateNo {
			// Validate seat count - API returns -1 when data is unavailable
			if loc.RemainSeatCnt < 0 {
				log.Printf("[Collector] Seat data not yet available for bus %s (got %d)", plateNo, loc.RemainSeatCnt)
				return nil
			}

			log.Printf("[Collector] Found bus %s at station seq %d, seats=%d",
				plateNo, loc.StationSeq, loc.RemainSeatCnt)
			seats := loc.RemainSeatCnt
			return &seats
		}
	}

	log.Printf("[Collector] Bus %s not found in location API results", plateNo)
	return nil
}

func (c *Collector) isWithinTimeWindow() bool {
	if c.startHour == 0 && c.endHour == 0 {
		return true // 24 hours
	}

	now := time.Now()
	hour := now.Hour()

	if c.startHour < c.endHour {
		return hour >= c.startHour && hour < c.endHour
	} else if c.startHour > c.endHour {
		// Cross-day: 22 to 2 means [22, 23, 0, 1]
		return hour >= c.startHour || hour < c.endHour
	}

	return hour == c.startHour
}
