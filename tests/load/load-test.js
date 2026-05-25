import http from 'k6/http';
import { check, sleep } from 'k6';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

export const options = {
  stages: [
    { duration: '10s', target: 50 },
    { duration: '30s', target: 500 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<100'],
    http_req_failed: ['rate<0.01'],
  },
  insecureSkipTlsVerify: true,
};

export default function () {
  const url = 'https://127.0.0.1:8443/healthz';
  const randomIP = `${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`;
  const params = {
    headers: {
      'Host': 'api.example.com',
      'X-Forwarded-For': randomIP,
    },
  };

  const res = http.get(url, params);
  
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
  
  sleep(0.1);
}

export function handleSummary(data) {
  return {
    "summary.html": htmlReport(data),
  };
}
