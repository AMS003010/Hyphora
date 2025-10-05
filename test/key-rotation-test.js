import http from "k6/http";
import { check, sleep } from "k6";

// k6 configuration
export const options = {
  vus: 10, // 10 virtual users
  duration: "30s", // Run for 30 seconds
  thresholds: {
    http_req_failed: ["rate<0.01"], // Error rate < 1%
    http_req_duration: ["p(95)<500"], // 95% of requests under 500ms
  },
};

// Generate a 1MB value
const VALUE_SIZE = 1024 * 1024; // 1MB
const VALUE = "x".repeat(VALUE_SIZE); // 1MB of 'x' characters

// Counter for unique keys
let keyCounter = 0;

// Main test function
export default function () {
  // Generate unique key for this VU and iteration
  const key = `key_${__VU}_${keyCounter++}`;

  // PUT request
  const putPayload = JSON.stringify({ key: key, value: VALUE });
  const putRes = http.post("http://192.168.0.9:8081/put", putPayload, {
    headers: { "Content-Type": "application/json" },
  });
  check(putRes, {
    "PUT succeeded": (r) => r.status === 204,
  });

  // GET request to verify
  const getRes = http.get(`http://192.168.0.10:8082/get?key=${key}`);
  check(getRes, {
    "GET succeeded": (r) => r.status === 200 && r.body.length === VALUE_SIZE,
  });

//   // Occasionally DELETE (e.g., 10% chance)
//   if (Math.random() < 0.1) {
//     const delPayload = JSON.stringify({ key: key });
//     const delRes = http.post("http://192.168.0.10:8081/del", delPayload, {
//       headers: { "Content-Type": "application/json" },
//     });
//     check(delRes, {
//       "DELETE succeeded": (r) => r.status === 204,
//     });
//   }

  // Small sleep to avoid overwhelming the server
  sleep(0.1);
}
