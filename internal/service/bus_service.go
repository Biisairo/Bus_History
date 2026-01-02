package service

import (
	"bus_history/internal/model"
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
)

// BusService provides unified access to both GBIS (Gyeonggi) and Incheon bus APIs
type BusService struct {
	gbisClient    *GBISClient
	incheonClient *IncheonClient
}

// NewBusService creates a new unified bus service
func NewBusService(gbisClient *GBISClient, incheonClient *IncheonClient) *BusService {
	return &BusService{
		gbisClient:    gbisClient,
		incheonClient: incheonClient,
	}
}

// SearchRoutes searches for routes in both Gyeonggi and Incheon
func (s *BusService) SearchRoutes(ctx context.Context, keyword string) ([]model.RouteInfo, error) {
	var allRoutes []model.RouteInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Search GBIS (Gyeonggi)
	wg.Add(1)
	go func() {
		defer wg.Done()
		routes, err := s.gbisClient.SearchRoutes(keyword)
		if err != nil {
			log.Printf("[BusService] GBIS route search error: %v", err)
			return
		}
		// Add region info
		for i := range routes {
			routes[i].RegionName = "경기 - " + routes[i].RegionName
		}
		mu.Lock()
		allRoutes = append(allRoutes, routes...)
		mu.Unlock()
	}()

	// Search Incheon
	wg.Add(1)
	go func() {
		defer wg.Done()
		routes, err := s.incheonClient.SearchRoutes(keyword)
		if err != nil {
			log.Printf("[BusService] Incheon route search error: %v", err)
			return
		}
		mu.Lock()
		allRoutes = append(allRoutes, routes...)
		mu.Unlock()
	}()

	wg.Wait()

	log.Printf("[BusService] Total routes found: %d", len(allRoutes))
	return allRoutes, nil
}

// SearchStations searches for stations in both Gyeonggi and Incheon
func (s *BusService) SearchStations(ctx context.Context, keyword string) ([]model.StationInfo, error) {
	var allStations []model.StationInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Search GBIS (Gyeonggi)
	wg.Add(1)
	go func() {
		defer wg.Done()
		stations, err := s.gbisClient.SearchStations(keyword)
		if err != nil {
			log.Printf("[BusService] GBIS station search error: %v", err)
			return
		}
		// Add region info
		for i := range stations {
			stations[i].RegionName = "경기 - " + stations[i].RegionName
		}
		mu.Lock()
		allStations = append(allStations, stations...)
		mu.Unlock()
	}()

	// Search Incheon
	wg.Add(1)
	go func() {
		defer wg.Done()
		stations, err := s.incheonClient.SearchStations(keyword)
		if err != nil {
			log.Printf("[BusService] Incheon station search error: %v", err)
			return
		}
		mu.Lock()
		allStations = append(allStations, stations...)
		mu.Unlock()
	}()

	wg.Wait()

	log.Printf("[BusService] Total stations found: %d", len(allStations))
	return allStations, nil
}

// GetRouteStations returns stations for a route from the appropriate API
func (s *BusService) GetRouteStations(ctx context.Context, routeID string, region string) ([]model.RouteStation, error) {
	if region == "인천" || region == "incheon" {
		return s.incheonClient.GetRouteStations(routeID)
	}
	// Default to GBIS
	return s.gbisClient.GetRouteStations(routeID)
}

// GetBusLocations returns bus locations for a route
func (s *BusService) GetBusLocations(ctx context.Context, routeID string, region string) ([]model.BusLocation, error) {
	if region == "인천" || region == "incheon" {
		// Incheon doesn't have a direct equivalent, return empty
		return []model.BusLocation{}, nil
	}
	return s.gbisClient.GetBusLocations(routeID)
}

// GetBusArrivalsByStation returns arrivals for a station
func (s *BusService) GetBusArrivalsByStation(ctx context.Context, stationID string, region string) ([]model.APIBusArrival, error) {
	if region == "인천" || region == "incheon" {
		return s.incheonClient.GetBusArrivalsByStation(stationID)
	}
	return s.gbisClient.GetBusArrivalsByStation(stationID)
}

// StationRouteInfo represents a route passing through a station
type StationRouteInfo struct {
	RouteID       int    `json:"routeId"`
	RouteName     string `json:"routeName"`
	RouteTypeName string `json:"routeTypeName"`
	Direction     string `json:"direction"` // 상행 or 하행
}

// GetStationRoutes returns routes passing through a station with direction info
func (s *BusService) GetStationRoutes(ctx context.Context, stationID string, region string) ([]StationRouteInfo, error) {
	if region == "인천" || region == "incheon" {
		// Fallback for Incheon: use arrivals since we don't have a direct routes-by-station API yet
		arrivals, err := s.incheonClient.GetBusArrivalsByStation(stationID)
		if err != nil {
			return nil, err
		}
		routeMap := make(map[string]StationRouteInfo)
		for _, a := range arrivals {
			id := fmt.Sprintf("%d", a.RouteID)
			if _, exists := routeMap[id]; !exists {
				routeMap[id] = StationRouteInfo{
					RouteID:       a.RouteID,
					RouteName:     a.RouteName,
					RouteTypeName: a.RouteTypeName,
					Direction:     a.Direction,
				}
			}
		}
		routes := make([]StationRouteInfo, 0, len(routeMap))
		for _, r := range routeMap {
			routes = append(routes, r)
		}
		return routes, nil
	}

	// GBIS: Use dedicated API
	gbisRoutes, err := s.gbisClient.GetRoutesByStation(stationID)
	if err != nil {
		return nil, err
	}

	result := make([]StationRouteInfo, 0, len(gbisRoutes))
	var wg sync.WaitGroup
	var mu sync.Mutex

	seenRoutes := make(map[int]bool)

	for _, r := range gbisRoutes {
		if seenRoutes[r.RouteID] {
			continue
		}
		seenRoutes[r.RouteID] = true

		wg.Add(1)
		go func(route model.RouteInfo) {
			defer wg.Done()

			direction := ""
			// Get station list for this route to find direction
			stations, err := s.gbisClient.GetRouteStations(fmt.Sprintf("%d", route.RouteID))
			if err == nil {
				currID, _ := strconv.Atoi(stationID)

				// Find turn point and current station position
				var turnSeq int = -1
				var currSeq int = -1

				for _, st := range stations {
					if st.TurnYn == "Y" {
						turnSeq = st.StationSeq
					}
					if st.StationID == currID {
						currSeq = st.StationSeq
					}
				}

				if currSeq != -1 {
					if turnSeq != -1 {
						if currSeq < turnSeq {
							direction = "상행"
						} else if currSeq == turnSeq {
							direction = "회차"
						} else {
							direction = "하행"
						}
					} else {
						// No turn point found, maybe it's a one-way?
						direction = "상행"
					}
				}
			}

			mu.Lock()
			result = append(result, StationRouteInfo{
				RouteID:       route.RouteID,
				RouteName:     route.RouteName,
				RouteTypeName: route.RouteTypeName,
				Direction:     direction,
			})
			mu.Unlock()
		}(r)
	}

	wg.Wait()
	return result, nil
}
