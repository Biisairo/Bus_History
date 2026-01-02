package model

import (
	"encoding/json"
	"fmt"
)

// RouteInfo represents bus route information
type RouteInfo struct {
	RouteID          int    `json:"routeId"`
	RouteName        string `json:"routeName"`
	RouteTypeName    string `json:"routeTypeName"`
	RouteTypeCd      int    `json:"routeTypeCd"`
	DistrictCd       int    `json:"districtCd"`
	StartStationID   int    `json:"startStationId"`
	StartStationName string `json:"startStationName"`
	EndStationID     int    `json:"endStationId"`
	EndStationName   string `json:"endStationName"`
	RegionName       string `json:"regionName"`
	AdminName        string `json:"adminName"`
}

// UnmarshalJSON custom unmarshaling to handle routeName as both string and number
func (r *RouteInfo) UnmarshalJSON(data []byte) error {
	type Alias RouteInfo
	aux := &struct {
		RouteName interface{} `json:"routeName"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch v := aux.RouteName.(type) {
	case string:
		r.RouteName = v
	case float64:
		r.RouteName = fmt.Sprintf("%.0f", v)
	case int:
		r.RouteName = fmt.Sprintf("%d", v)
	default:
		r.RouteName = fmt.Sprintf("%v", v)
	}

	return nil
}

// StationInfo represents bus station information
type StationInfo struct {
	StationID   int     `json:"stationId"`
	StationName string  `json:"stationName"`
	RegionName  string  `json:"regionName"`
	DistrictCd  int     `json:"districtCd"`
	CenterYn    string  `json:"centerYn"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	MobileNo    string  `json:"mobileNo"`
}

// RouteStation represents a station on a route
type RouteStation struct {
	StationID   int     `json:"stationId"`
	StationName string  `json:"stationName"`
	StationSeq  int     `json:"stationSeq"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	TurnYn      string  `json:"turnYn"`
	RegionName  string  `json:"regionName"`
}

// BusLocation represents current bus location
type BusLocation struct {
	RouteID       int    `json:"routeId"`
	StationID     int    `json:"stationId"`
	StationSeq    int    `json:"stationSeq"`
	PlateNo       string `json:"plateNo"`
	PlateType     int    `json:"plateType"`
	RemainSeatCnt int    `json:"remainSeatCnt"`
	StationName   string `json:"stationName"`
}

// APIBusArrival represents bus arrival information from Gyeonggi/Incheon API
type APIBusArrival struct {
	RouteID       int    `json:"routeId"`
	RouteName     string `json:"routeName"`
	RouteTypeName string `json:"routeTypeName"`
	StationID     int    `json:"stationId"`
	StationSeq    int    `json:"staOrder"`
	PlateNo       string `json:"plateNo"`
	RemainSeatCnt int    `json:"remainSeatCnt"`
	PredictTime1  int    `json:"predictTime1"`
	LocationNo1   int    `json:"locationNo1"`
	LowPlate1     int    `json:"lowPlate1"`
	Direction     string `json:"direction"` // 상행 or 하행
}

// BusArrivalInfo represents bus arrival information from the OpenAPI
type BusArrivalInfo struct {
	RouteID       int    `json:"routeId"`
	StationID     int    `json:"stationId"`
	StationSeq    int    `json:"staOrder"`
	PlateNo       string `json:"plateNo"`
	RemainSeatCnt int    `json:"remainSeatCnt"`
	PredictTime1  int    `json:"predictTime1"`
	LocationNo1   int    `json:"locationNo1"`
	LowPlate1     int    `json:"lowPlate1"`
}
