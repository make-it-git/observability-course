import http from 'k6/http';

export const options = {
    stages: [
        { duration: '10m', target: 100 },
    ],
};

const URL = 'http://localhost:8080/process';

export default function () {
    http.get(URL);
}