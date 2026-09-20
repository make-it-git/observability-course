import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '5s', target: 20 },   // Ramp up to 20 virtual users over 5s seconds
        { duration: '10m', target: 20 },    // Stay at 20 virtual users for 10 minutes
        { duration: '5s', target: 0 },    // Ramp down to 0 virtual users over 5 seconds
    ],
    thresholds: {
        'http_req_duration': ['p(90)<500'], // 90% of requests should be below 500ms
        'http_req_failed': ['rate<0.1'],    // Error rate should be less than 10%
    },
};

const rideRequestURL = 'http://localhost:8080/request-ride';

export default function () {
    const payload = JSON.stringify({
        pickup: 'locationA',
        dropoff: 'locationB',
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const res = http.post(rideRequestURL, payload, params);

    check(res, {
        'is status 200': (r) => r.status === 200,
        'response time is acceptable': (r) => r.timings.duration < 500, // Optional: add a response time check
    });

    sleep(1); // Add a pause to simulate user think time
}