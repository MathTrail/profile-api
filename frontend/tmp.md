mathtrail-profile/
├── backend/                 👈 Go backend (изолирован)
│   ├── cmd/
│   │   └── profile-api/
│   │       └── main.go
│   ├── internal/
│   ├── pkg/
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile
│   └── README.md
│
├── frontend/                👈 ВСЁ про frontend
│   ├── apps/
│   │   └── profile/         👈 microfrontend
│   │       ├── src/
│   │       ├── public/
│   │       ├── index.html
│   │       ├── vite.config.ts
│   │       ├── tailwind.config.ts
│   │       └── package.json
│   │
│   ├── packages/
│   │   └── ui/              👈 @project/ui
│   │       ├── src/
│   │       ├── tailwind.config.ts
│   │       └── package.json
│   │
│   ├── package.json         👈 workspace root
│   ├── tsconfig.base.json
│   └── tailwind.base.config.ts
│
├── infra/                   👈 k8s / helm / terraform (опционально)
│
├── .devcontainer/
│
├── .gitignore
├── README.md