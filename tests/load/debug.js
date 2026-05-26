import http from 'k6/http';

export const options = {
  vus: 50,
  duration: '5s',
  insecureSkipTlsVerify: true,
};

export default function () {
  http.get('https://127.0.0.1:8443/healthz', {
    headers: { 'Host': 'api.example.com' }
  });
}
