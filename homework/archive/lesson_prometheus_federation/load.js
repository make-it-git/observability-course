import http from 'k6/http';
import { check, sleep } from 'k6';

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

export default function () {
    const id = Math.floor(Math.random() * 30000) + 1;
    const res1 = http.get(`http://localhost:8080/get-user/${id}`);
    const res2 = http.post(`http://localhost:8081/send-message/${id}`, 'dummy date');
    check(res1, {
        'is status 200': (r) => r.status === 200,
        'response time is acceptable': (r) => r.timings.duration < 500,
    });
    check(res2, {
        'is status 200': (r) => r.status === 200,
        'response time is acceptable': (r) => r.timings.duration < 500,
    });
}