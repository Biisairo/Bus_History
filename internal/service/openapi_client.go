package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"bus_history/internal/model"
)

// OpenAPIClient handles communication with the public data API (used by collector)
type OpenAPIClient struct {
	baseURL    string
	serviceKey string
	client     *http.Client
}

// NewOpenAPIClient creates a new API client
func NewOpenAPIClient(baseURL, serviceKey string) *OpenAPIClient {
	return &OpenAPIClient{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetBusArrivalList retrieves bus arrival information for a station
func (c *OpenAPIClient) GetBusArrivalList(stationID string) ([]model.BusArrivalInfo, error) {
	endpoint := "https://apis.data.go.kr/6410000/busarrivalservice/v2/getBusArrivalListv2"

	params := url.Values{}
	params.Add("serviceKey", c.serviceKey)
	params.Add("stationId", stationID)
	params.Add("format", "json")

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.URL.RawQuery = params.Encode()
	req.Header.Set("User-Agent", "Mozilla/5.0")

	log.Printf("[OpenAPI] Requesting: %s", req.URL.String())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
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

	var arrivals []model.BusArrivalInfo
	if err := json.Unmarshal(jsonResp.Response.MsgBody.BusArrivalList, &arrivals); err != nil {
		var singleArrival model.BusArrivalInfo
		if err := json.Unmarshal(jsonResp.Response.MsgBody.BusArrivalList, &singleArrival); err != nil {
			return []model.BusArrivalInfo{}, nil
		}
		arrivals = []model.BusArrivalInfo{singleArrival}
	}

	return arrivals, nil
}

// GetRouteArrivalList retrieves bus arrival information for a specific route at a station
func (c *OpenAPIClient) GetRouteArrivalList(routeID, stationID string) ([]model.BusArrivalInfo, error) {
	endpoint := "https://apis.data.go.kr/6410000/busarrivalservice/v2/getBusArrivalItemv2"

	params := url.Values{}
	params.Add("serviceKey", c.serviceKey)
	params.Add("routeId", routeID)
	params.Add("stationId", stationID)
	params.Add("format", "json")

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.URL.RawQuery = params.Encode()
	req.Header.Set("User-Agent", "Mozilla/5.0")

	log.Printf("[OpenAPI] Requesting: %s", req.URL.String())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var jsonResp struct {
		Response struct {
			MsgHeader struct {
				ResultCode int    `json:"resultCode"`
				ResultMsg  string `json:"resultMessage"`
			} `json:"msgHeader"`
			MsgBody struct {
				BusArrivalItem struct {
					PlateNo1       string `json:"plateNo1"`
					PlateNo2       string `json:"plateNo2"`
					PredictTime1   int    `json:"predictTime1"`
					PredictTime2   int    `json:"predictTime2"`
					LocationNo1    int    `json:"locationNo1"`
					LocationNo2    int    `json:"locationNo2"`
					RemainSeatCnt1 int    `json:"remainSeatCnt1"`
					RemainSeatCnt2 int    `json:"remainSeatCnt2"`
					LowPlate1      int    `json:"lowPlate1"`
					LowPlate2      int    `json:"lowPlate2"`
					RouteID        int    `json:"routeId"`
					StationID      int    `json:"stationId"`
				} `json:"busArrivalItem"`
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

	var arrivals []model.BusArrivalInfo
	item := jsonResp.Response.MsgBody.BusArrivalItem

	if item.PlateNo1 != "" {
		arrivals = append(arrivals, model.BusArrivalInfo{
			RouteID:       item.RouteID,
			StationID:     item.StationID,
			PlateNo:       item.PlateNo1,
			PredictTime1:  item.PredictTime1,
			LocationNo1:   item.LocationNo1,
			RemainSeatCnt: item.RemainSeatCnt1,
			LowPlate1:     item.LowPlate1,
		})
	}

	if item.PlateNo2 != "" {
		arrivals = append(arrivals, model.BusArrivalInfo{
			RouteID:       item.RouteID,
			StationID:     item.StationID,
			PlateNo:       item.PlateNo2,
			PredictTime1:  item.PredictTime2,
			LocationNo1:   item.LocationNo2,
			RemainSeatCnt: item.RemainSeatCnt2,
			LowPlate1:     item.LowPlate2,
		})
	}

	return arrivals, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
