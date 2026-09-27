# Nodo frontend

This frontend has no JavaScript build step or package dependencies. In Docker, Nginx serves the files and proxies `/api` to the backend container.

The normal way to run it is from the repository root:

```powershell
Copy-Item .env.example .env
# Add OPENROUTER_API_KEY to .env
docker compose up --build
```

Open `http://localhost:5173`. The frontend receives `NODO_API_URL=/api`, while Nginx uses `API_UPSTREAM=http://api:8080` on Docker's private network.

For development without Docker, create `frontend/config.js` before serving the directory:

```javascript
window.NODO_CONFIG = { apiUrl: "http://localhost:8080" };
```
