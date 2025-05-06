import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Trend } from 'k6/metrics';

export const options = {
    stages: [
        { duration: '30s', target: 100 }, // ramp-up до 100 VU за 30 сек
        { duration: '1m', target: 100 },  // стабильная нагрузка 100 VU
        { duration: '30s', target: 0 },   // ramp-down до 0 VU за 30 сек
    ],
    thresholds: {
        'update_driver_location_duration': ['p(95)<200'], // 95-й процентиль времени ответа < 200 мс
        'find_nearest_drivers_duration': ['p(95)<300'], // 95-й процентиль времени ответа < 300 мс
        'http_req_duration{endpoint:update_driver_location}': ['p(95)<200'], // 95-й процентиль времени ответа < 200 мс
        'http_req_duration{endpoint:find_nearest_drivers}': ['p(95)<300'], // 95-й процентиль времени ответа < 300 мс

        'http_req_failed': ['rate<0.01'],   // процент ошибок < 1%
        'http_reqs': ['rate>100'], // Количество запросов в секунду > 100

    },
};

const summaryTrend = {
    'update_driver_location_duration': new Trend('update_driver_location_duration'),
    'find_nearest_drivers_duration': new Trend('find_nearest_drivers_duration'),
};

const BASE_URL = 'http://localhost:8080';
const TOTAL_DRIVERS = 10000;

function getRandomDriverId() {
    return Math.floor(Math.random() * TOTAL_DRIVERS) + 1;
}

function getRandomCoordinates() {
    const lat = 55.0 + Math.random() * 2.0; // Примерный диапазон координат Москвы
    const lon = 37.0 + Math.random() * 2.0;
    return { lat: lat.toFixed(6), lon: lon.toFixed(6) };
}


export default function () {
    group('Update Driver Location', () => {
        const driverId = getRandomDriverId();
        const { lat, lon } = getRandomCoordinates();
        const url = `${BASE_URL}/drivers/update?driver_id=${driverId}&lat=${lat}&lon=${lon}`;

        let res = http.get(url, {
            tags: { endpoint: 'update_driver_location'},
        });

        check(res, {
            'Update location status is 200': (r) => r.status === 200,
        });

        summaryTrend['update_driver_location_duration'].add(res.timings.duration);

        sleep(1); // Пауза между запросами обновления координат
    });


    group('Find Nearest Drivers', () => {
        const { lat, lon } = getRandomCoordinates();
        const limit = 5;
        const url = `${BASE_URL}/drivers/search?lat=${lat}&lon=${lon}&limit=${limit}`;

        let res = http.get(url, {
            tags: { endpoint: 'find_nearest_drivers'},
        });

        check(res, {
            'Find drivers status is 200': (r) => r.status === 200,
            'Response contains drivers': (r) => r.body.includes('drivers'),
        });
        summaryTrend['find_nearest_drivers_duration'].add(res.timings.duration);

        sleep(2);
    });
}


export function handleSummary(data) {
    console.log('Performance Summary:\n');

    for(const [key, value] of Object.entries(summaryTrend)) {
        console.log(`\t${key}:`);
        console.log(`\t\tavg: ${data.metrics[key].values.avg}`);
        console.log(`\t\tmax: ${data.metrics[key].values.max}`);
        console.log(`\t\tp95: ${data.metrics[key].values['p(95)']}`);
    }
}
