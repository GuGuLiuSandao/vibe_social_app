export default {
  testRunner: "vitest",
  mutate: ["src/lib/uid.js"],
  reporters: ["clear-text", "json"],
  jsonReporter: { fileName: "../quality/frontend-mutation.json" },
  thresholds: { high: 100, low: 100, break: 100 },
  timeoutMS: 10000,
  concurrency: 2,
};
