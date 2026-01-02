// State
let selectedRoute = null;
let selectedStation = null;
let currentMode = 'route-first'; // 'route-first' or 'station-first'
let currentViewedConfig = null; // Currently viewed config for auto-refresh
let isCollecting = false;

// Initialization
document.addEventListener('DOMContentLoaded', async () => {
	setupEnterKey();
	initApp();
});

async function initApp() {
	try {
		const settings = await window.go.main.App.GetSettings();
		if (!settings || !settings.storagePath || !settings.serviceKey) {
			showView('options');
			showNotification('시스템 설정을 먼저 완료해주세요.', 'error');
		} else {
			// Apply settings to UI
			document.getElementById('storage-path').value = settings.storagePath || '';
			document.getElementById('api-key').value = settings.serviceKey || '';
			document.getElementById('start-hour').value = settings.startHour || 0;
			document.getElementById('end-hour').value = settings.endHour || 0;
			document.getElementById('interval-ms').value = settings.intervalMs || 30000;

			// Initial status check
			updateCollectionStatus();
			showView('home');
		}
	} catch (e) {
		console.error("Init failed", e);
		showView('options');
	}
}

function lockSettings(locked) {
	document.getElementById('api-key').disabled = locked;
	document.getElementById('start-hour').disabled = locked;
	document.getElementById('end-hour').disabled = locked;
	document.getElementById('interval-ms').disabled = locked;
	const submitBtn = document.querySelector('#options-view .submit-btn');
	if (submitBtn) {
		submitBtn.disabled = locked;
		submitBtn.title = locked ? "수집 중에는 설정을 변경할 수 없습니다." : "";
	}
}

// View Management
function showView(viewName) {
	// Switch active view
	document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
	document.getElementById(`${viewName}-view`).classList.add('active');

	// Update active tab in nav
	document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
	const activeBtn = document.getElementById(`nav-${viewName}`);
	if (activeBtn) activeBtn.classList.add('active');

	// Load data if needed
	if (viewName === 'list') {
		loadConfigs();
	}
}

// Collection Control
async function toggleCollection() {
	try {
		const status = await window.go.main.App.GetCollectionStatus();
		if (status && isCollecting) {
			await window.go.main.App.StopCollection();
			showNotification('수집이 중지되었습니다.');
		} else {
			await window.go.main.App.StartCollection();
			showNotification('수집이 시작되었습니다!', 'success');
		}
		updateCollectionStatus();
	} catch (e) {
		showNotification('수집 제어 실패: ' + e, 'error');
	}
}

async function updateCollectionStatus() {
	try {
		const status = await window.go.main.App.GetCollectionStatus();
		isCollecting = status;
		const btn = document.getElementById('main-toggle-btn');
		const title = btn.querySelector('.menu-title');

		if (status) {
			btn.classList.add('collecting');
			title.textContent = '수집 중지 (작동중)';
		} else {
			btn.classList.remove('collecting');
			title.textContent = '수집 시작';
		}
		lockSettings(status);
	} catch (e) { }
}

// Settings
async function selectStoragePath() {
	try {
		const path = await window.go.main.App.SelectFolder();
		if (path) {
			document.getElementById('storage-path').value = path;
		}
	} catch (e) {
		showNotification('폴더 선택 실패: ' + e, 'error');
	}
}

async function saveSettings() {
	const path = document.getElementById('storage-path').value.trim();
	const key = document.getElementById('api-key').value.trim();
	const startHour = parseInt(document.getElementById('start-hour').value) || 0;
	const endHour = parseInt(document.getElementById('end-hour').value) || 0;
	const intervalMs = parseInt(document.getElementById('interval-ms').value) || 30000;

	if (!path || !key) {
		showNotification('모든 항목을 입력하세요', 'error');
		return;
	}

	try {
		await window.go.main.App.UpdateSettings(path, key, startHour, endHour, intervalMs);
		showNotification('설정이 저장 및 적용되었습니다!', 'success');
		showView('home');
	} catch (e) {
		showNotification('설정 저장 실패: ' + e, 'error');
	}
}

// Search Logic (Selection Mode Switch)
function switchMode(mode) {
	currentMode = mode;
	document.querySelectorAll('.mode-btn').forEach(btn => btn.classList.remove('active'));
	document.querySelectorAll('.selection-mode').forEach(m => m.classList.remove('active'));
	document.querySelector(`[data-mode="${mode}"]`).classList.add('active');
	document.getElementById(`${mode}-mode`).classList.add('active');
	resetSelection();
}

function resetSelection() {
	selectedRoute = null;
	selectedStation = null;
	updateSummary();
	updateRegisterButton();
}

function setupEnterKey() {
	document.getElementById('rf-route-keyword')?.addEventListener('keypress', e => {
		if (e.key === 'Enter') searchRoutesForRouteFirst();
	});
	document.getElementById('sf-station-keyword')?.addEventListener('keypress', e => {
		if (e.key === 'Enter') searchStationsForStationFirst();
	});
}

function showNotification(message, type = 'success') {
	const notification = document.getElementById('notification');
	notification.textContent = message;
	notification.className = `notification ${type}`;
	notification.classList.remove('hidden');
	setTimeout(() => notification.classList.add('hidden'), 3000);
}

// --- Data Bindings Implementation ---

async function searchRoutesForRouteFirst() {
	const keyword = document.getElementById('rf-route-keyword').value.trim();
	if (!keyword) return;

	try {
		const results = await window.go.main.App.SearchRoutes(keyword);
		const resultsDiv = document.getElementById('rf-route-results');

		if (!results || results.length === 0) {
			resultsDiv.innerHTML = '<div class="empty">검색 결과가 없습니다</div>';
			return;
		}

		resultsDiv.innerHTML = results.map((route, idx) => `
            <div class="result-item" onclick="selectRouteForRouteFirst(${idx})">
                <div class="result-name">${route.routeName}</div>
                <div class="result-type">${route.routeTypeName || ''}</div>
                <div class="result-region">${route.regionName || ''}</div>
            </div>
        `).join('');

		// Store results globally for indexing
		window._routeSearchResults = results;
	} catch (e) {
		showNotification('검색 실패: ' + e, 'error');
	}
}

async function selectRouteForRouteFirst(idx) {
	const route = window._routeSearchResults[idx];
	selectedRoute = route;
	selectedStation = null;

	document.getElementById('rf-route-selected').innerHTML = `<strong>선택됨:</strong> ${route.routeName}`;
	document.getElementById('rf-route-selected').classList.remove('hidden');

	const region = route.regionName?.includes('인천') ? '인천' : '경기';
	const stations = await window.go.main.App.GetRouteStations(String(route.routeId), region);

	const resultsDiv = document.getElementById('rf-station-results');
	document.getElementById('rf-station-hint').style.display = 'none';

	const labeledStations = addDirectionLabels(stations);
	resultsDiv.innerHTML = labeledStations.map((s, idx) => `
        <div class="result-item" onclick="selectStationForRouteFirst(${idx})">
            <div class="result-name">${s.displayName}</div>
        </div>
    `).join('');

	window._stationSearchResults = labeledStations;
	updateSummary();
	updateRegisterButton();
}

function selectStationForRouteFirst(idx) {
	selectedStation = window._stationSearchResults[idx];
	document.getElementById('rf-station-selected').innerHTML = `<strong>선택됨:</strong> ${selectedStation.displayName}`;
	document.getElementById('rf-station-selected').classList.remove('hidden');
	updateSummary();
	updateRegisterButton();
}

async function searchStationsForStationFirst() {
	const keyword = document.getElementById('sf-station-keyword').value.trim();
	if (!keyword) return;

	try {
		const results = await window.go.main.App.SearchStations(keyword);
		const resultsDiv = document.getElementById('sf-station-results');

		if (!results || results.length === 0) {
			resultsDiv.innerHTML = '<div class="empty">검색 결과가 없습니다</div>';
			return;
		}

		resultsDiv.innerHTML = results.map((s, idx) => `
            <div class="result-item" onclick="selectStationForStationFirst(${idx})">
                <div class="result-name">${s.stationName}</div>
                <div class="result-region">${s.regionName || ''}</div>
            </div>
        `).join('');

		window._stationSearchResults = results;
	} catch (e) {
		showNotification('검색 실패: ' + e, 'error');
	}
}

async function selectStationForStationFirst(idx) {
	const station = window._stationSearchResults[idx];
	selectedStation = station;
	selectedRoute = null;

	document.getElementById('sf-station-selected').innerHTML = `<strong>선택됨:</strong> ${station.stationName}`;
	document.getElementById('sf-station-selected').classList.remove('hidden');

	const region = station.regionName?.includes('인천') ? '인천' : '경기';
	const routes = await window.go.main.App.GetStationRoutes(String(station.stationId), region);

	const resultsDiv = document.getElementById('sf-route-results');
	document.getElementById('sf-route-hint').style.display = 'none';

	resultsDiv.innerHTML = routes.map((r, idx) => `
        <div class="result-item" onclick="selectRouteForStationFirst(${idx})">
            <div class="result-name">${r.routeName} ${r.direction ? `(${r.direction})` : ''}</div>
        </div>
    `).join('');

	window._routeSearchResults = routes;
	updateSummary();
	updateRegisterButton();
}

function selectRouteForStationFirst(idx) {
	selectedRoute = window._routeSearchResults[idx];
	document.getElementById('sf-route-selected').innerHTML = `<strong>선택됨:</strong> ${selectedRoute.routeName}`;
	document.getElementById('sf-route-selected').classList.remove('hidden');
	updateSummary();
	updateRegisterButton();
}

function updateSummary() {
	document.getElementById('summary-route').textContent = selectedRoute ? selectedRoute.routeName : '노선을 선택해주세요';
	document.getElementById('summary-station').textContent = selectedStation ? selectedStation.displayName : '정류장을 선택해주세요';
}

function updateRegisterButton() {
	document.getElementById('register-btn').disabled = !(selectedRoute && selectedStation);
}

async function registerMonitoring() {
	try {
		await window.go.main.App.CreateConfig({
			route_id: String(selectedRoute.routeId),
			route_name: selectedRoute.routeName,
			station_id: String(selectedStation.stationId),
			station_name: selectedStation.stationName,
			direction: selectedStation.direction || selectedRoute.direction || '',
			sta_order: selectedStation.stationSeq || 0
		});
		showNotification('등록되었습니다!', 'success');
		showView('list');
	} catch (e) {
		showNotification('등록 실패: ' + e, 'error');
	}
}

async function loadConfigs() {
	const listDiv = document.getElementById('configs-content');
	try {
		const configs = await window.go.main.App.GetConfigs();
		if (!configs || configs.length === 0) {
			listDiv.innerHTML = '<div class="empty">등록된 모니터링이 없습니다</div>';
			return;
		}

		listDiv.innerHTML = `
            <table>
                <thead><tr><th>노선</th><th>정류장 (방향)</th><th>상태</th><th>작업</th></tr></thead>
                <tbody>
                    ${configs.map(c => `
                        <tr class="clickable-row" onclick="viewArrivals(${c.id}, '${c.route_id}', '${c.station_id}', '${c.route_name}', '${c.station_name}')">
                            <td>${c.route_name}</td>
                            <td>${c.station_name} ${c.direction ? `(${c.direction})` : ''}</td>
                            <td>${c.is_active ? '✅' : '❌'}</td>
                            <td>
                                <button onclick="event.stopPropagation(); toggleConfig(${c.id}, ${!c.is_active})">토글</button>
                                <button onclick="event.stopPropagation(); deleteConfig(${c.id})">삭제</button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
            <div id="selected-config-arrivals"></div>
        `;
	} catch (e) {
		listDiv.innerHTML = '로드 실패: ' + e;
	}
}

async function viewArrivals(id, routeId, stationId, routeName, stationName) {
	currentViewedConfig = { id, routeId, stationId, routeName, stationName };
	refreshCurrentArrivals();
}

async function refreshCurrentArrivals() {
	if (!currentViewedConfig) return;
	const { routeId, stationId, routeName, stationName } = currentViewedConfig;
	const div = document.getElementById('selected-config-arrivals');
	const date = document.getElementById('global-date').value;

	try {
		const result = await window.go.main.App.GetArrivals(routeId, stationId, date, date, 1, 50);
		if (!result || !result.data || result.data.length === 0) {
			div.innerHTML = `<h3>📊 ${routeName} 도착 이력</h3><div class="empty">지정한 날짜에 수집된 도착 정보가 없습니다.</div>`;
			return;
		}

		div.innerHTML = `
            <h3>📊 ${routeName} 도착 이력</h3>
            <table>
                <thead><tr><th>차량</th><th>도착시간</th><th>도착시</th><th>출발시</th><th>탑승</th></tr></thead>
                <tbody>
                    ${result.data.map(a => `
                        <tr class="clickable-row" onclick="viewTripDetail(${a.id})">
                            <td>${a.bus_number}</td>
                            <td>${new Date(a.arrival_time).toLocaleTimeString()}</td>
                            <td>${a.seats_before ?? '-'}</td>
                            <td>${a.seats_after ?? '-'}</td>
                            <td><strong>${(a.seats_before ?? 0) - (a.seats_after ?? 0)}명</strong></td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
            <div id="trip-detail"></div>
        `;
	} catch (e) {
		div.innerHTML = '이력 로드 실패: ' + e;
	}
}

async function viewTripDetail(id) {
	const div = document.getElementById('trip-detail');
	try {
		const trip = await window.go.main.App.GetTrip(id);
		if (!trip || trip.length === 0) return;

		div.innerHTML = `
            <div class="trip-detail-container">
				<div class="trip-header">
					<h4>🚌 회차 상세 (차량: ${trip[0].bus_number})</h4>
					<p>노선: ${trip[0].route_name} | 수집 시간: ${new Date(trip[0].arrival_time).toLocaleTimeString()}</p>
				</div>
				<div class="trip-timeline">
					${trip.map(t => `
						<div class="timeline-item ${t.id === id ? 'target' : ''}">
							<div class="timeline-marker"></div>
							<div class="timeline-content">
								<div class="timeline-station">${t.station_name}</div>
								<div class="timeline-boarding">
									<strong>${(t.seats_before ?? 0) - (t.seats_after ?? 0)}명</strong> 탑승
									<span class="timeline-seats">(${t.seats_before} ➔ ${t.seats_after})</span>
								</div>
							</div>
						</div>
					`).join('')}
				</div>
            </div>
        `;
	} catch (e) {
		div.innerHTML = '회차 정보 실패: ' + e;
	}
}

async function toggleConfig(id, active) {
	await window.go.main.App.ToggleConfig(id, active);
	loadConfigs();
}

async function deleteConfig(id) {
	if (confirm('삭제하시겠습니까?')) {
		await window.go.main.App.DeleteConfig(id);
		loadConfigs();
	}
}

function clearDateFilter() {
	document.getElementById('global-date').value = '';
	refreshCurrentArrivals();
}

function addDirectionLabels(stations) {
	// 1. find index of turn point
	const turnIndex = stations.findIndex(s => s.turnYn === 'Y');
	if (turnIndex === -1) {
		return stations.map(s => ({ ...s, displayName: s.stationName, direction: "" }));
	}

	return stations.map((s, idx) => {
		let direction = "";
		if (idx < turnIndex) {
			direction = "상행";
		} else if (idx === turnIndex) {
			direction = "회차";
		} else {
			direction = "하행";
		}
		return {
			...s,
			displayName: `${s.stationName} (${direction})`,
			direction: direction
		};
	});
}
