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

// IncheonClient handles communication with the Incheon Bus API
type IncheonClient struct {
	serviceKey string
	client     *http.Client
}

// NewIncheonClient creates a new Incheon Bus API client
func NewIncheonClient(serviceKey string) *IncheonClient {
	return &IncheonClient{
		serviceKey: serviceKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ============================================================================
// Helper Methods
// ============================================================================

func (c *IncheonClient) makeRequest(endpoint string, params url.Values) ([]byte, error) {
	params.Add("serviceKey", c.serviceKey)
	params.Add("pageNo", "1")
	params.Add("numOfRows", "100")
	params.Add("_type", "json")

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.URL.RawQuery = params.Encode()
	req.Header.Set("User-Agent", "Mozilla/5.0")

	log.Printf("[Incheon] Requesting URL: %s", req.URL.String())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[Incheon] API returned non-200 status: %d, Body: %s", resp.StatusCode, string(bodyBytes))
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

// IncheonRouteInfo represents raw route info from Incheon API
type IncheonRouteInfo struct {
	RouteID   string `json:"ROUTEID"`
	RouteNo   string `json:"ROUTENO"`
	RouteType string `json:"ROUTETPCD"`
	StartStop string `json:"ORIGIN_BSTOPNM"`
	EndStop   string `json:"DEST_BSTOPNM"`
}

// SearchRoutes searches for bus routes by keyword
func (c *IncheonClient) SearchRoutes(keyword string) ([]model.RouteInfo, error) {
	endpoint := "https://apis.data.go.kr/6280000/busRouteInfo/getRouteNoList"
	params := url.Values{}
	params.Add("routeNo", keyword)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var jsonResp struct {
		Response struct {
			Header struct {
				ResultCode string `json:"resultCode"`
				ResultMsg  string `json:"resultMsg"`
			} `json:"header"`
			Body struct {
				Items struct {
					Item json.RawMessage `json:"item"`
				} `json:"items"`
			} `json:"body"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.Header.ResultCode != "00" {
		return nil, fmt.Errorf("API error (code %s): %s",
			jsonResp.Response.Header.ResultCode,
			jsonResp.Response.Header.ResultMsg)
	}

	// Handle both array and single object cases
	var incheonRoutes []IncheonRouteInfo
	if err := json.Unmarshal(jsonResp.Response.Body.Items.Item, &incheonRoutes); err != nil {
		var singleRoute IncheonRouteInfo
		if err := json.Unmarshal(jsonResp.Response.Body.Items.Item, &singleRoute); err != nil {
			// No results
			return []model.RouteInfo{}, nil
		}
		incheonRoutes = []IncheonRouteInfo{singleRoute}
	}

	// Convert to common RouteInfo format
	routes := make([]model.RouteInfo, len(incheonRoutes))
	for i, r := range incheonRoutes {
		routeID := 0
		fmt.Sscanf(r.RouteID, "%d", &routeID)
		routes[i] = model.RouteInfo{
			RouteID:          routeID,
			RouteName:        r.RouteNo,
			RouteTypeName:    c.getRouteTypeName(r.RouteType),
			RegionName:       "인천",
			StartStationName: r.StartStop,
			EndStationName:   r.EndStop,
		}
	}

	log.Printf("[Incheon] Successfully parsed %d routes", len(routes))
	return routes, nil
}

func (c *IncheonClient) getRouteTypeName(typeCode string) string {
	switch typeCode {
	case "1":
		return "일반"
	case "2":
		return "좌석"
	case "3":
		return "급행"
	case "4":
		return "광역"
	case "5":
		return "마을"
	default:
		return "일반"
	}
}

// ============================================================================
// Station Service APIs
// ============================================================================

// IncheonStationInfo represents raw station info from Incheon API
type IncheonStationInfo struct {
	StationID   string  `json:"BSTOPID"`
	StationName string  `json:"BSTOPNM"`
	PosX        float64 `json:"POSX"`
	PosY        float64 `json:"POSY"`
	ShortID     string  `json:"SHORT_BSTOPID"`
}

// SearchStations searches for bus stations by keyword
func (c *IncheonClient) SearchStations(keyword string) ([]model.StationInfo, error) {
	endpoint := "https://apis.data.go.kr/6280000/busStationInfo/getBstopInfoList"
	params := url.Values{}
	params.Add("bstopNm", keyword)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	var jsonResp struct {
		Response struct {
			Header struct {
				ResultCode string `json:"resultCode"`
				ResultMsg  string `json:"resultMsg"`
			} `json:"header"`
			Body struct {
				Items struct {
					Item json.RawMessage `json:"item"`
				} `json:"items"`
			} `json:"body"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.Header.ResultCode != "00" {
		return nil, fmt.Errorf("API error (code %s): %s",
			jsonResp.Response.Header.ResultCode,
			jsonResp.Response.Header.ResultMsg)
	}

	var incheonStations []IncheonStationInfo
	if err := json.Unmarshal(jsonResp.Response.Body.Items.Item, &incheonStations); err != nil {
		var singleStation IncheonStationInfo
		if err := json.Unmarshal(jsonResp.Response.Body.Items.Item, &singleStation); err != nil {
			return []model.StationInfo{}, nil
		}
		incheonStations = []IncheonStationInfo{singleStation}
	}

	stations := make([]model.StationInfo, len(incheonStations))
	for i, s := range incheonStations {
		stationID := 0
		fmt.Sscanf(s.StationID, "%d", &stationID)
		stations[i] = model.StationInfo{
			StationID:   stationID,
			StationName: s.StationName,
			RegionName:  "인천",
			X:           s.PosX,
			Y:           s.PosY,
			MobileNo:    s.ShortID,
		}
	}

	return stations, nil
}

// IncheonRouteStation represents a station on a route from Incheon API
type IncheonRouteStation struct {
	StationID   string `json:"BSTOPID"`
	StationName string `json:"BSTOPNM"`
	StationSeq  int    `json:"BSTOPSEQ"`
}

// GetRouteStations gets all stations on a route
func (c *IncheonClient) GetRouteStations(routeID string) ([]model.RouteStation, error) {
	endpoint := "https://apis.data.go.kr/6280000/busRouteInfo/getRouteBstopList"
	params := url.Values{}
	params.Add("routeId", routeID)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	var jsonResp struct {
		Response struct {
			Header struct {
				ResultCode string `json:"resultCode"`
				ResultMsg  string `json:"resultMsg"`
			} `json:"header"`
			Body struct {
				Items struct {
					Item json.RawMessage `json:"item"`
				} `json:"items"`
			} `json:"body"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.Header.ResultCode != "00" {
		return nil, fmt.Errorf("API error (code %s): %s",
			jsonResp.Response.Header.ResultCode,
			jsonResp.Response.Header.ResultMsg)
	}

	var incheonStations []IncheonRouteStation
	if err := json.Unmarshal(jsonResp.Response.Body.Items.Item, &incheonStations); err != nil {
		var singleStation IncheonRouteStation
		if err := json.Unmarshal(jsonResp.Response.Body.Items.Item, &singleStation); err != nil {
			return []model.RouteStation{}, nil
		}
		incheonStations = []IncheonRouteStation{singleStation}
	}

	stations := make([]model.RouteStation, len(incheonStations))
	for i, s := range incheonStations {
		stationID := 0
		fmt.Sscanf(s.StationID, "%d", &stationID)
		stations[i] = model.RouteStation{
			StationID:   stationID,
			StationName: s.StationName,
			StationSeq:  s.StationSeq,
			RegionName:  "인천",
		}
	}

	return stations, nil
}

// IncheonArrival represents arrival info from Incheon API
type IncheonArrival struct {
	RouteID       string `json:"ROUTEID"`
	RouteName     string `json:"ROUTENO"`
	ArrivalTime   int    `json:"ARRIVALESTIMATETIME"` // seconds
	RestStopCount int    `json:"REST_STOP_COUNT"`
	PlateNo       string `json:"BUS_NUM_PLATE"`
	RemainSeatCnt int    `json:"REMAINSEATCNT"`
}

func (c *IncheonClient) GetBusArrivalList(stationID string) ([]model.APIBusArrival, error) {
	endpoint := "https://apis.data.go.kr/6280000/busArrInfo/getStaionArrInfo"
	params := url.Values{}
	params.Add("bstopId", stationID)

	body, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	var jsonResp struct {
		Response struct {
			Header struct {
				ResultCode string `json:"resultCode"`
				ResultMsg  string `json:"resultMsg"`
			} `json:"header"`
			Body struct {
				Items struct {
					Item json.RawMessage `json:"item"`
				} `json:"items"`
			} `json:"body"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if jsonResp.Response.Header.ResultCode != "00" {
		return nil, fmt.Errorf("API error (code %s): %s",
			jsonResp.Response.Header.ResultCode,
			jsonResp.Response.Header.ResultMsg)
	}

	var incheonArrivals []IncheonArrival
	if err := json.Unmarshal(jsonResp.Response.Body.Items.Item, &incheonArrivals); err != nil {
		var singleArrival IncheonArrival
		if err := json.Unmarshal(jsonResp.Response.Body.Items.Item, &singleArrival); err != nil {
			return []model.APIBusArrival{}, nil
		}
		incheonArrivals = []IncheonArrival{singleArrival}
	}

	arrivals := make([]model.APIBusArrival, len(incheonArrivals))
	for i, a := range incheonArrivals {
		routeID := 0
		fmt.Sscanf(a.RouteID, "%d", &routeID)
		stID := 0
		fmt.Sscanf(stationID, "%d", &stID)

		arrivals[i] = model.APIBusArrival{
			RouteID:       routeID,
			StationID:     stID,
			PredictTime1:  a.ArrivalTime / 60, // Convert seconds to minutes
			LocationNo1:   a.RestStopCount,
			PlateNo:       a.PlateNo,
			RemainSeatCnt: a.RemainSeatCnt,
		}
	}

	return arrivals, nil
}

// GetBusArrivalsByStation is an alias for GetBusArrivalList to match interface
func (c *IncheonClient) GetBusArrivalsByStation(stationID string) ([]model.APIBusArrival, error) {
	return c.GetBusArrivalList(stationID)
}
