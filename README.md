<div align="center">

# 🎁 Gift

### **Your money. Your server. Your crew.**

**A self-hosted, multi-user spendings & incomes tracker built for groups who actually split things — roommates, couples, travel crews, small teams.**

[![Go](https://img.shields.io/badge/Go-1.25.6-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Vue 3](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vuedotjs&logoColor=white)](https://vuejs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![MongoDB](https://img.shields.io/badge/MongoDB-v2-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com)
[![Fiber](https://img.shields.io/badge/Fiber-v3-00ACD7)](https://gofiber.io)
[![GoReleaser](https://img.shields.io/badge/GoReleaser-✓-5D5D5D?logo=goreleaser&logoColor=white)](https://goreleaser.com)
[![Release](https://img.shields.io/github/v/release/aslon1213/gift?include_prereleases&sort=semver)](https://github.com/aslon1213/gift/releases)

[🚀 Quick Start](#-quick-start) · [✨ Features](#-features) · [📦 Install](#-install) · [🛠️ Stack](#️-tech-stack) · [📚 API Docs](#-api-docs) · [🤝 Contributing](#-contributing)

</div>

---

## 🔥 Why Gift?

Ever tried to split a trip with 4 friends using a spreadsheet? **Painful.** Tried a SaaS tracker that suddenly wants $9/mo and owns your data? **Worse.**

**Gift is the middle path:** spin it up on your own box, invite your people, and start tracking. No subscriptions, no data sharing, no surprises. Cross-platform binaries for Linux, macOS and Windows — `amd64` and `arm64` — built on every tag.

> 💡 **Self-host in 5 minutes.** Invite your crew in 5 seconds. Track forever.

---

## ✨ Features

### 👥 Groups — the core primitive
Create a **group** for anything you split: *"Bali 2026"*, *"Apartment 3B"*, *"Sunday poker"*. Invite users, track shared expenses against it, see who's contributing what.

- Create / rename / delete groups
- Owner-managed membership — invite by user search, remove members anytime
- A spending can be **personal** *or* **group-scoped** — one unified API, one unified feed
- Query "groups I own" vs "groups I'm a member of" independently

### 💸 Spendings
Log expenses with the fields that actually matter.

| Field         | Notes                                    |
| ------------- | ---------------------------------------- |
| `amount`      | Decimal, any precision                   |
| `currency`    | Multi-currency ready                     |
| `category`    | Free-form, powers the donut chart        |
| `description` | Because "Starbucks $14" deserves context |
| `date`        | Time-range queries supported             |
| `group_id`    | Optional — omit for personal spending    |

Filter by **user, group, category, date range** with pagination (`limit` / `offset`) baked into the API.

### 💰 Incomes
Salary, freelance, dividends, a tip jar — log it, see it, net it against spendings.

### 🎯 Budgets, Goals, Alerts
- **Budgets** — cap spending per category / group / period
- **Goals** — "Save $3k for Bali by August" — progress-tracked
- **Alerts** — opinionated nudges when things go sideways

### 🌐 Bring-your-own-server UX

- First thing the web app asks: **"Where's your server?"** Point it at `localhost`, a Tailscale IP, a domain — whatever. No hardcoded backend. One frontend build can talk to any Gift instance.
---

## 🖼️ Screenshots
<div align="center">

| Dashboard | Groups | Spendings |
| :-: | :-: | :-: |
| ![Dashboard](docs/screenshots/dashboard.png) | ![Groups](docs/screenshots/groups.png) | ![Spendings](docs/screenshots/spendings.png) |

</div>

---

## 🚀 Quick Start

### Prerequisites
- [**mise**](https://mise.jdx.dev/getting-started.html) (handles Go, Node, air, swag, etc. automatically)
- **MongoDB** running somewhere reachable — [local](https://www.mongodb.com/docs/manual/installation/), [Atlas](https://www.mongodb.com/atlas), or a container:
  ```sh
  docker run -d --name gift-mongo -p 27017:27017 mongo:latest
  ```

### 1. Clone & install tools
```sh
git clone https://github.com/aslon1213/gift.git
cd gift
mise install   # pulls Go 1.26, Node 25, air, swag, goreleaser, golangci-lint…
```

### 2. Configure the server
Create `server/.env`:
```dotenv
DB_URL=mongodb://localhost:27017
DB_AUTH=false
AUTH_JWT_SECRET=change-me-to-a-long-random-string
AUTH_JWT_REFRESH_SECRET=change-me-too
AUTH_JWT_EXPIRES_IN=15m
AUTH_JWT_REFRESH_EXPIRES_IN=720h
```

### 3. Run it 🏃
Open two terminals:

```sh
# Terminal 1 — API on :3000 (hot-reloading)
mise run api:dev
```
```sh
# Terminal 2 — Vue dev server on :5173 (proxies /api → :3000)
mise run web:dev
```

Open **http://localhost:5173**, point the server setup at `http://localhost:3000`, register your first user — **you're now the admin of your own finance tracker.** 🎉

---

## 📦 Install

### Grab a release binary
```sh
# Pick your platform from the releases page
curl -L -o gift-api.tar.gz \
  https://github.com/aslon1213/gift/releases/latest/download/gift-api_<VERSION>_Linux_x86_64.tar.gz
tar -xzf gift-api.tar.gz
./gift-api
```

Releases include:
- `gift-api_*` — server binary for Linux / macOS / Windows, `amd64` + `arm64`
- `gift-web_*` — static frontend bundle (serve with Nginx, Caddy, S3, anywhere)
- `docs/swagger.json` + `docs/swagger.yaml` — OpenAPI spec
- `gift_*_checksums.txt` — SHA256 sums

### Or Docker
```sh
docker build -f deployment/Dockerfile.server -t gift-api .
docker build -f deployment/Dockerfile.web    -t gift-web .
```

### Or build from source
```sh
mise run api:build     # → server/bin/api_gateway
cd client/gift-web && npm ci && npm run build   # → client/gift-web/dist
```

---

## 🤝 Contributing

PRs welcome

1. Fork + branch
2. `mise install && mise run api:dev` — make sure it runs locally
3. Install pre-commit hooks: `pre-commit install`
4. Open a PR with a clear description

Found a bug? [Open an issue.](https://github.com/aslon1213/gift/issues)

---

<div align="center">

**Built with ☕, Go, and a healthy suspicion of subscription finance apps.**

⭐ Star the repo if Gift saved you from a spreadsheet.

</div>
