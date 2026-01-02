# Bus History Collection System - Frontend

경기도 버스 정보 수집 시스템의 웹 인터페이스

## 기술 스택

- **프레임워크**: Svelte + TypeScript
- **빌드 도구**: Vite
- **스타일**: Vanilla CSS

## 시작하기

### 1. 의존성 설치

```bash
npm install
```

### 2. 환경 변수 설정

`.env` 파일을 생성하고 백엔드 API 주소를 설정하세요:

```bash
# .env.example 파일을 복사
cp .env.example .env

# 필요시 API 주소 수정
VITE_API_BASE=http://localhost:8080/api
```

### 3. 개발 서버 실행

```bash
npm run dev
```

브라우저에서 http://localhost:5173 열기

### 4. 프로덕션 빌드

```bash
npm run build
```

## 주요 기능

### 📝 모니터링 등록
- 노선 검색 및 선택
- 정류장 검색 및 선택
- 모니터링 대상 등록

### ⚙️ 모니터링 목록
- 등록된 모니터링 설정 확인
- 활성화/비활성화 전환
- 모니터링 삭제

### 📊 데이터 조회
- 수집된 버스 도착 정보 조회
- 노선/정류장/날짜 필터링
- 페이지네이션

## 컴포넌트 구조

```
src/
├── lib/
│   ├── components/
│   │   ├── RouteSearch.svelte        # 노선 검색
│   │   ├── StationSearch.svelte      # 정류장 검색
│   │   ├── MonitoringForm.svelte     # 모니터링 등록 폼
│   │   ├── ConfigList.svelte         # 설정 목록
│   │   └── ArrivalData.svelte        # 도착 데이터 조회
│   ├── api.ts                        # API 클라이언트
│   ├── stores.ts                     # Svelte 상태 관리
│   └── types.ts                      # TypeScript 타입 정의
├── App.svelte                        # 메인 앱 컴포넌트
└── main.ts                           # 진입점
```

## API 사용 예시

```typescript
import { lookupAPI, configAPI, arrivalAPI } from './lib/api';

// 노선 검색
const routes = await lookupAPI.searchRoutes('M5100');

// 정류장 검색
const stations = await lookupAPI.searchStations('강남역');

// 모니터링 등록
await configAPI.create({
  route_id: '233000031',
  station_id: '228000719',
  station_name: '강남역'
});

// 데이터 조회
const arrivals = await arrivalAPI.getArrivals({
  route_id: '233000031',
  from_date: '2025-12-01',
  page: 1,
  limit: 20
});
```

## 개발 가이드

### 새 컴포넌트 추가

1. `src/lib/components/` 에 `.svelte` 파일 생성
2. TypeScript 타입은 `src/lib/types.ts`에 정의
3. API 호출은 `src/lib/api.ts` 사용
4. 상태 관리는 `src/lib/stores.ts` 활용

### 스타일링

- 각 컴포넌트의 `<style>` 블록에 작성
- Scoped CSS 자동 적용
- 전역 스타일은 `App.svelte`의 `:global()` 사용

## 백엔드 연동

백엔드 서버가 실행 중이어야 합니다:

```bash
cd ../backend
go run cmd/server/main.go
```

백엔드가 http://localhost:8080에서 실행되면 프론트엔드와 자동 연동됩니다.

## 빌드 및 배포

```bash
# 프로덕션 빌드
npm run build

# 빌드 미리보기
npm run preview
```

빌드 결과는 `dist/` 폴더에 생성됩니다.

## 라이선스

MIT License
