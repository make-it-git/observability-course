import { sleep } from 'k6';
import http from 'k6/http';

export const options = {
  iterations: 1_000_000,
};

export default function () {
  const payload = JSON.stringify({ item_id: 101, quantity: 1 });
  const params = { headers: { 'Content-Type': 'application/json' } };
  
  http.post('http://app:8080/api/v1/orders', payload, params);
  sleep(0.1);
}
