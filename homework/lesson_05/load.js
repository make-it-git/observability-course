import http from 'k6/http';
import { check } from 'k6';

export const options = {
    scenarios: {
        constant_rps: {
            executor: 'constant-arrival-rate',
            duration: '10m',
            rate: 100,
            gracefulStop: '0s',
            preAllocatedVUs: 100,
        },
    },
    thresholds: {
        'http_req_duration': ['p(90)<500'], // 90% of requests should be below 500ms
        'http_req_failed': ['rate<0.1'],    // Error rate should be less than 10%
    },
};

const countryCodes = [
    'US', 'CA', 'GB', 'DE', 'FR', 'JP', 'AU',
    'BR', 'CN', 'IN', 'MX', 'RU', 'KR', 'IT',
    'ES', 'NL', 'SE', 'CH', 'BE', 'NO', 'DK',
    'PL', 'AT', 'FI', 'IE', 'AR', 'CL', 'CO',
    'ZA', 'NG', 'KE', 'EG', 'SA', 'AE', 'SG',
    'HK', 'TW', 'ID', 'MY', 'PH', 'TH', 'VN',
];
function getRandomRegion() {
    const randomIndex = Math.floor(Math.random() * countryCodes.length);
    return countryCodes[randomIndex];
}

export default function () {
    const params = {
        headers: {
            'X-Region': getRandomRegion(),
        },
    };
    const id = Math.floor(Math.random() * 50000) + 1;
    const res = http.get(`http://localhost:8080/user/${id}`, );
    check(res, {
        'is status 200': (r) => r.status === 200,
        'response time is acceptable': (r) => r.timings.duration < 500,
    });
}