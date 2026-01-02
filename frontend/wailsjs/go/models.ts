export namespace config {
	
	export class AppSettings {
	    storagePath: string;
	    serviceKey: string;
	    startHour: number;
	    endHour: number;
	    intervalMs: number;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.storagePath = source["storagePath"];
	        this.serviceKey = source["serviceKey"];
	        this.startHour = source["startHour"];
	        this.endHour = source["endHour"];
	        this.intervalMs = source["intervalMs"];
	    }
	}

}

export namespace model {
	
	export class BusArrivalWithConfig {
	    id: number;
	    route_config_id: number;
	    bus_number: string;
	    // Go type: time
	    arrival_time: any;
	    seats_before?: number;
	    seats_after?: number;
	    // Go type: time
	    created_at: any;
	    route_id: string;
	    route_name: string;
	    station_id: string;
	    station_name: string;
	    sta_order: number;
	
	    static createFrom(source: any = {}) {
	        return new BusArrivalWithConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.route_config_id = source["route_config_id"];
	        this.bus_number = source["bus_number"];
	        this.arrival_time = this.convertValues(source["arrival_time"], null);
	        this.seats_before = source["seats_before"];
	        this.seats_after = source["seats_after"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.route_id = source["route_id"];
	        this.route_name = source["route_name"];
	        this.station_id = source["station_id"];
	        this.station_name = source["station_name"];
	        this.sta_order = source["sta_order"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RouteConfig {
	    id: number;
	    route_id: string;
	    route_name: string;
	    station_id: string;
	    station_name: string;
	    direction: string;
	    sta_order: number;
	    is_active: boolean;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new RouteConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.route_id = source["route_id"];
	        this.route_name = source["route_name"];
	        this.station_id = source["station_id"];
	        this.station_name = source["station_name"];
	        this.direction = source["direction"];
	        this.sta_order = source["sta_order"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RouteInfo {
	    routeId: number;
	    routeName: string;
	    routeTypeName: string;
	    routeTypeCd: number;
	    districtCd: number;
	    startStationId: number;
	    startStationName: string;
	    endStationId: number;
	    endStationName: string;
	    regionName: string;
	    adminName: string;
	
	    static createFrom(source: any = {}) {
	        return new RouteInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.routeId = source["routeId"];
	        this.routeName = source["routeName"];
	        this.routeTypeName = source["routeTypeName"];
	        this.routeTypeCd = source["routeTypeCd"];
	        this.districtCd = source["districtCd"];
	        this.startStationId = source["startStationId"];
	        this.startStationName = source["startStationName"];
	        this.endStationId = source["endStationId"];
	        this.endStationName = source["endStationName"];
	        this.regionName = source["regionName"];
	        this.adminName = source["adminName"];
	    }
	}
	export class RouteStation {
	    stationId: number;
	    stationName: string;
	    stationSeq: number;
	    x: number;
	    y: number;
	    turnYn: string;
	    regionName: string;
	
	    static createFrom(source: any = {}) {
	        return new RouteStation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stationId = source["stationId"];
	        this.stationName = source["stationName"];
	        this.stationSeq = source["stationSeq"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.turnYn = source["turnYn"];
	        this.regionName = source["regionName"];
	    }
	}
	export class StationInfo {
	    stationId: number;
	    stationName: string;
	    regionName: string;
	    districtCd: number;
	    centerYn: string;
	    x: number;
	    y: number;
	    mobileNo: string;
	
	    static createFrom(source: any = {}) {
	        return new StationInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stationId = source["stationId"];
	        this.stationName = source["stationName"];
	        this.regionName = source["regionName"];
	        this.districtCd = source["districtCd"];
	        this.centerYn = source["centerYn"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.mobileNo = source["mobileNo"];
	    }
	}

}

export namespace service {
	
	export class StationRouteInfo {
	    routeId: number;
	    routeName: string;
	    routeTypeName: string;
	    direction: string;
	
	    static createFrom(source: any = {}) {
	        return new StationRouteInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.routeId = source["routeId"];
	        this.routeName = source["routeName"];
	        this.routeTypeName = source["routeTypeName"];
	        this.direction = source["direction"];
	    }
	}

}

