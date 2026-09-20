import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
  vus: 5,
  duration: '300m',
};

export default function () {
  http.get('http://app:8080/orders');
  sleep(0.05);
}
