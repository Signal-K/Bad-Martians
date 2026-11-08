// Runtime configuration, loaded before host.js.
//   api: location.origin            the page's own PocketBase server (what `make run` serves)
//   api: ""                         demo mode, no backend: progress stays in this browser, sign-in hidden
//   api: "https://backend.example"  another PocketBase backend (it must allow this origin via CORS)
// scripts/build-dist.sh rewrites this file for static hosting from the HM_API_URL
// environment variable, empty by default. It holds a public URL only, never a secret.
window.HM_CONFIG = { api: location.origin };
