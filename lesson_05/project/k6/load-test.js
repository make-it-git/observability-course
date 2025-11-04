import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';

// Configuration
const BASE_URL = 'http://localhost:8080';
const DRIVERS_COUNT = 100;
const UPDATE_INTERVAL = 5; // seconds
const BERLIN_CENTER = {
    lat: 52.52,
    lon: 13.405
};

// Speed configuration (km/h)
const MIN_SPEED = 30;
const MAX_SPEED = 60;

// Earth's radius in kilometers
const EARTH_RADIUS = 6371;

// Initial driver data (read-only)
const initialDrivers = new SharedArray('drivers', function () {
    const arr = [];
    for (let i = 0; i < DRIVERS_COUNT; i++) {
        arr.push({
            id: i,
            lat: BERLIN_CENTER.lat + (Math.random() * 0.1 - 0.05), // ±5km spread
            lon: BERLIN_CENTER.lon + (Math.random() * 0.1 - 0.05),
            speed: MIN_SPEED + Math.random() * (MAX_SPEED - MIN_SPEED), // 30-60 km/h
            heading: Math.random() * 2 * Math.PI, // Random initial heading
            lastUpdate: Date.now()
        });
    }
    return arr;
});

// Mutable driver states
let driverStates = [];

export const options = {
    scenarios: {
        constant_load: {
            executor: 'constant-vus',
            vus: 10,
            duration: '50m',
        },
    },
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
        'http_req_duration{type:location_update}': ['p(95)<400'],
        'http_req_duration{type:nearby_search}': ['p(95)<600'],
    },
};

// Convert degrees to radians
function toRadians(degrees) {
    return degrees * (Math.PI / 180);
}

// Convert radians to degrees
function toDegrees(radians) {
    return radians * (180 / Math.PI);
}

// Calculate new position based on current position, speed, heading and time
function calculateNewPosition(driver, timeDiff) {
    const hours = timeDiff / 3600; // Convert seconds to hours
    const distance = driver.speed * hours; // Distance in kilometers

    // Convert current position to radians
    const lat1 = toRadians(driver.lat);
    const lon1 = toRadians(driver.lon);
    const bearing = driver.heading;

    // Calculate new position
    const lat2 = Math.asin(
        Math.sin(lat1) * Math.cos(distance / EARTH_RADIUS) +
        Math.cos(lat1) * Math.sin(distance / EARTH_RADIUS) * Math.cos(bearing)
    );

    const lon2 = lon1 + Math.atan2(
        Math.sin(bearing) * Math.sin(distance / EARTH_RADIUS) * Math.cos(lat1),
        Math.cos(distance / EARTH_RADIUS) - Math.sin(lat1) * Math.sin(lat2)
    );

    // Convert back to degrees
    return {
        latitude: toDegrees(lat2),
        longitude: toDegrees(lon2)
    };
}

function updateDriverLocation(driverState) {
    const now = Date.now();
    const timeSinceLastUpdate = (now - driverState.lastUpdate) / 1000; // Convert to seconds
    
    // Skip update if not enough time has passed
    if (timeSinceLastUpdate < UPDATE_INTERVAL) {
        return;
    }
    
    // Calculate new position
    const location = calculateNewPosition(driverState, UPDATE_INTERVAL); // Use fixed interval
    
    // Update stored position for next iteration
    driverState.lat = location.latitude;
    driverState.lon = location.longitude;
    driverState.lastUpdate = now;

    // Slightly adjust heading occasionally (10% chance)
    if (Math.random() < 0.1) {
        driverState.heading += (Math.random() * 0.5 - 0.25); // ±0.25 radians (±14 degrees)
    }

    const payload = JSON.stringify(location);
    const headers = { 'Content-Type': 'application/json' };
    
    const response = http.post(
        `${BASE_URL}/api/v1/drivers/${driverState.id}/location`,
        payload,
        { 
            headers,
            tags: { type: 'location_update' }
        }
    );

    check(response, {
        'location update successful': (r) => r.status === 200,
    });
}

function searchNearbyDrivers() {
    // Random point in Berlin area
    const searchLat = BERLIN_CENTER.lat + (Math.random() * 0.1 - 0.05);
    const searchLon = BERLIN_CENTER.lon + (Math.random() * 0.1 - 0.05);
    
    const response = http.get(
        `${BASE_URL}/api/v1/drivers/nearby?lat=${searchLat}&lon=${searchLon}&radius=2000`,
        { 
            tags: { type: 'nearby_search' }
        }
    );

    check(response, {
        'nearby search successful': (r) => r.status === 200,
    });
}

export default function () {
    // Initialize driver states on first run
    if (driverStates.length === 0) {
        // Stagger initial lastUpdate times to spread out updates
        driverStates = initialDrivers.map((driver, index) => ({
            ...driver,
            lastUpdate: Date.now() - (index * (UPDATE_INTERVAL * 1000 / DRIVERS_COUNT))
        }));
    }

    // Try to update all drivers - only those that need updating will be processed
    for (let driverState of driverStates) {
        updateDriverLocation(driverState);
    }

    // 20% chance to perform a nearby search
    if (Math.random() < 0.2) {
        searchNearbyDrivers();
    }

    // Sleep for a shorter interval to check more frequently
    sleep(1);
} 