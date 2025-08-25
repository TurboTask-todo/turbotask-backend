import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '1s', target: 1 }, // Ramp-up to 10k RPS
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be < 500ms
    http_req_failed: ['rate<0.01'],   // Errors should be < 1%
  },
};

const url = 'http://localhost:8080/api/v1/todo/projects'; // Replace with your actual URL

export default function () {
  const payload = JSON.stringify({
    title: "Mobile App Development",
    description: "Building a cross-platform mobile application",
    category: "work",
    color_theme: "#2ecc71",
    image_url: "https://example.com/project-image.jpg"
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiZDAzNjRhZTgtNjgyZS00YzFiLTkzY2UtZmNkYmZmNjZjODU1IiwiZW1haWwiOiJ1c2VyQGV4YW1wbGUuY29tIiwidXNlcm5hbWUiOiJ0ZXN0dXNlciIsInRva2VuX3R5cGUiOiJhY2Nlc3MiLCJzZXNzaW9uX2lkIjoiZjQ0Y2RlYjUtNDZlMi00MzQzLThjYmQtN2FhOGYwNmY5MzcwIiwiaXNzIjoiTWFjV3JpdGUgQXV0aCBBUEkiLCJzdWIiOiJkMDM2NGFlOC02ODJlLTRjMWItOTNjZS1mY2RiZmY2NmM4NTUiLCJhdWQiOlsibWFjd3JpdGUtYXBpIl0sImV4cCI6MTc1NDM5ODcxMywibmJmIjoxNzU0Mzk3ODEzLCJpYXQiOjE3NTQzOTc4MTMsImp0aSI6IjMyYzg1MGYzLTBmYTgtNDY0Zi1iNGYyLWViYjRjZDg3ZDNkMyJ9.Uvf5UeXusWxAge1JUJXaCjBZ816JN7I_PloY0OA7FFY' // Optional auth
    },
  };

  const res = http.post(url, payload, params);

  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
