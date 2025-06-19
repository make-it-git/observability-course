import http from 'k6/http';

export const options = {
    stages: [
        { duration: '5s', target: 200 },
        { duration: '10m', target: 1000 },
        { duration: '5s', target: 0 },
    ],
};

const URL = 'http://localhost:8080/process';

export default function () {
    http.get(URL);
}