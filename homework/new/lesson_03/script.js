import http from 'k6/http';

export const options = {
  scenarios: {
    constant_request_rate: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '300m',
      preAllocatedVUs: 100,
    },
  },
};

export default function () {
  http.get('http://app:8080/api/v1/cart');
  // Executed ~50% of the time
  if (Math.random() < 0.5) {
    http.get('http://app:8080/api/v1/checkout');
  }
}
