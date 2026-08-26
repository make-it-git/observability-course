import http from 'k6/http';

export const options = {
    scenarios: {
        constant_rps: {
            executor: 'constant-arrival-rate',
            duration: '10m',
            rate: 100,
            gracefulStop: '0s',
            preAllocatedVUs: 100,
        },
    }
};

const errorPercentage = 5;
const errorProbability = errorPercentage / 100;

export default function () {
    http.get(`http://localhost:8080/ok`);
    if (Math.random() < errorProbability) {
        http.get('http://localhost:8080/error');
    }
}