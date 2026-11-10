// Runtime configuration, loaded before host.js.
//   api: location.origin            the page's own PocketBase server (what `make run` serves)
//   api: ""                         demo mode, no backend: progress stays in this browser, sign-in hidden
//   api: "https://backend.example"  another PocketBase backend (it must allow this origin via CORS)
//   clerk: "pk_live_..."            Clerk publishable key: sign in with the Star Sailors account
//                                   through Clerk and the backend's /auth/clerk-exchange
// scripts/build-dist.sh rewrites this file for static hosting from the HM_API_URL and
// HM_CLERK_KEY environment variables, empty by default. Both are public, never secrets.
window.HM_CONFIG = { api: location.origin };
