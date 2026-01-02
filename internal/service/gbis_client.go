package service

import (
	"bus_history/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// GBISClient handles communication with the GBIS API for all bus services
type GBISClient struct {
	serviceKey string
	client     *http.Client
}

// NewGBISClient creates a new GBIS API client
func NewGBISClient(serviceKey string) *GBISClient {
	return &GBISClient{
		serviceKey: serviceKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ============================================================================
// Helper Methods
// ============================================================================

func (c *GBISClient) makeRequest(endpoint string, params url.Values) ([]byte, error) {
	params.Add("serviceKey", c.serviceKey)
	params.Add("format", "json")

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.URL.RawQuery = params.Encode()
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	log.Printf("Requesting URL: %s", req.URL.String())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("API returned non-200 status: %d, Body: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// ============================================================================
// Route Service APIs
// ============================================================================

// SearchRoutes searches for bus routes by keyword
func (c *GBISClient) SearchRoutes(keyword string) ([]model.RouteInfo, error) {
	endpoint := "https://apis.data.go.kr/6410000/busrouteservice/v2/getBusRouteListv2"
	params := url.Values{}
	params.Add("keyword", keyword)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	var jsonResp struct {
		Response struct {
			MsgHeader struct {
				ResultCode int    `json:"resultCode"`
				ResultMsg  string `json:"resultMessage"`
			} `json:"msgHeader"`
			MsgBody struct {
				BusRouteList json.RawMessage `json:"busRouteList"`
			} `json:"msgBody"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.MsgHeader.ResultCode != 0 {
		return nil, fmt.Errorf("API error (code %d): %s",
			jsonResp.Response.MsgHeader.ResultCode,
			jsonResp.Response.MsgHeader.ResultMsg)
	}

	var routes []model.RouteInfo
	if err := json.Unmarshal(jsonResp.Response.MsgBody.BusRouteList, &routes); err != nil {
		var singleRoute model.RouteInfo
		if err := json.Unmarshal(jsonResp.Response.MsgBody.BusRouteList, &singleRoute); err != nil {
			return []model.RouteInfo{}, nil
		}
		routes = []model.RouteInfo{singleRoute}
	}

	return routes, nil
}

// GetRouteStations gets all stations on a route
func (c *GBISClient) GetRouteStations(routeID string) ([]model.RouteStation, error) {
	endpoint := "https://apis.data.go.kr/6410000/busrouteservice/v2/getBusRouteStationListv2"
	params := url.Values{}
	params.Add("routeId", routeID)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	var jsonResp struct {
		Response struct {
			MsgHeader struct {
				ResultCode int    `json:"resultCode"`
				ResultMsg  string `json:"resultMessage"`
			} `json:"msgHeader"`
			MsgBody struct {
				BusRouteStationList []model.RouteStation `json:"busRouteStationList"`
			} `json:"msgBody"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.MsgHeader.ResultCode != 0 {
		return nil, fmt.Errorf("API error (code %d): %s",
			jsonResp.Response.MsgHeader.ResultCode,
			jsonResp.Response.MsgHeader.ResultMsg)
	}

	return jsonResp.Response.MsgBody.BusRouteStationList, nil
}

// ============================================================================
// Station Service APIs
// ============================================================================

// SearchStations searches for bus stations by keyword
func (c *GBISClient) SearchStations(keyword string) ([]model.StationInfo, error) {
	endpoint := "https://apis.data.go.kr/6410000/busstationservice/v2/getBusStationListv2"
	params := url.Values{}
	params.Add("keyword", keyword)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	var jsonResp struct {
		Response struct {
			MsgHeader struct {
				ResultCode int    `json:"resultCode"`
				ResultMsg  string `json:"resultMessage"`
			} `json:"msgHeader"`
			MsgBody struct {
				BusStationList json.RawMessage `json:"busStationList"`
			} `json:"msgBody"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.MsgHeader.ResultCode != 0 {
		return nil, fmt.Errorf("API error (code %d): %s",
			jsonResp.Response.MsgHeader.ResultCode,
			jsonResp.Response.MsgHeader.ResultMsg)
	}

	var stations []model.StationInfo
	if err := json.Unmarshal(jsonResp.Response.MsgBody.BusStationList, &stations); err != nil {
		var singleStation model.StationInfo
		if err := json.Unmarshal(jsonResp.Response.MsgBody.BusStationList, &singleStation); err != nil {
			return []model.StationInfo{}, nil
		}
		stations = []model.StationInfo{singleStation}
	}

	return stations, nil
}

// ============================================================================
// Location Service APIs
// ============================================================================

// GetBusLocations gets current bus locations on a route
func (c *GBISClient) GetBusLocations(routeID string) ([]model.BusLocation, error) {
	endpoint := "https://apis.data.go.kr/6410000/buslocationservice/v2/getBusLocationListv2"
	params := url.Values{}
	params.Add("routeId", routeID)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	var jsonResp struct {
		Response struct {
			MsgHeader struct {
				ResultCode int    `json:"resultCode"`
				ResultMsg  string `json:"resultMessage"`
			} `json:"msgHeader"`
			MsgBody struct {
				BusLocationList []model.BusLocation `json:"busLocationList"`
			} `json:"msgBody"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.MsgHeader.ResultCode != 0 {
		return nil, fmt.Errorf("API error (code %d): %s",
			jsonResp.Response.MsgHeader.ResultCode,
			jsonResp.Response.MsgHeader.ResultMsg)
	}

	return jsonResp.Response.MsgBody.BusLocationList, nil
}

// ============================================================================
// Arrival Service APIs
// ============================================================================

func (c *GBISClient) GetBusArrivalsByStation(stationID string) ([]model.APIBusArrival, error) {
	endpoint := "https://apis.data.go.kr/6410000/busarrivalservice/v2/getBusArrivalListv2"
	params := url.Values{}
	params.Add("stationId", stationID)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	var jsonResp struct {
		Response struct {
			MsgHeader struct {
				ResultCode int    `json:"resultCode"`
				ResultMsg  string `json:"resultMessage"`
			} `json:"msgHeader"`
			MsgBody struct {
				BusArrivalList json.RawMessage `json:"busArrivalList"`
			} `json:"msgBody"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.MsgHeader.ResultCode != 0 {
		return nil, fmt.Errorf("API error (code %d): %s",
			jsonResp.Response.MsgHeader.ResultCode,
			jsonResp.Response.MsgHeader.ResultMsg)
	}

	var arrivals []model.APIBusArrival
	if err := json.Unmarshal(jsonResp.Response.MsgBody.BusArrivalList, &arrivals); err != nil {
		var singleArrival model.APIBusArrival
		if err := json.Unmarshal(jsonResp.Response.MsgBody.BusArrivalList, &singleArrival); err != nil {
			return []model.APIBusArrival{}, nil
		}
		arrivals = []model.APIBusArrival{singleArrival}
	}

	return arrivals, nil
}

// GetRoutesByStation gets all bus routes passing through a station
func (c *GBISClient) GetRoutesByStation(stationID string) ([]model.RouteInfo, error) {
	endpoint := "https://apis.data.go.kr/6410000/busstationservice/v2/getBusStationViaRouteListv2"
	params := url.Values{}
	params.Add("stationId", stationID)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	var jsonResp struct {
		Response struct {
			MsgHeader struct {
				ResultCode int    `json:"resultCode"`
				ResultMsg  string `json:"resultMessage"`
			} `json:"msgHeader"`
			MsgBody struct {
				BusRouteList json.RawMessage `json:"busRouteList"`
			} `json:"msgBody"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.MsgHeader.ResultCode != 0 {
		if jsonResp.Response.MsgHeader.ResultCode == 3 {
			return []model.RouteInfo{}, nil
		}
		return nil, fmt.Errorf("API error (code %d): %s",
			jsonResp.Response.MsgHeader.ResultCode,
			jsonResp.Response.MsgHeader.ResultMsg)
	}

	var routes []model.RouteInfo
	if err := json.Unmarshal(jsonResp.Response.MsgBody.BusRouteList, &routes); err != nil {
		var singleRoute model.RouteInfo
		if err := json.Unmarshal(jsonResp.Response.MsgBody.BusRouteList, &singleRoute); err != nil {
			return []model.RouteInfo{}, nil
		}
		routes = []model.RouteInfo{singleRoute}
	}

	return routes, nil
}
