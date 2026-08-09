# TypstLab Server

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![Chi](https://img.shields.io/badge/Chi-008080?style=flat&logo=go&logoColor=white)](https://github.com/go-chi/chi)

**TypstLab Server** is the server-side component of the TypstLab collaborative editing platform. Built with **Go**, it acts as the central coordinator for real-time collaboration, document state synchronization (using CRDTs), and data persistence.

For the client-side application, see the [frontend repository](https://github.com/safarislava/typstlab-app) (or equivalent frontend repo).

---

## 🏗 Backend Architecture & Responsibilities

The backend is designed to be lightweight, high-performance, and secure, focusing on the following core domains:

1. **CRDT Synchronization**: 
   - Acts as the central synchronization authority using **Yjs** protocol adapters written in Go.
   - Receives, validates, merges, and broadcasts conflict-free document update chunks.
2. **Real-time Communication**:
   - Manages persistent **WebSocket** connections for active collaborative sessions.
   - Dispatches user presence, cursors, and system notifications in real-time.
3. **Data Persistence**:
   - Stores user profiles, workspace metadata, and compressed document state updates in ???Db.
   - Provides point-in-time recovery and snapshot generation for documents.
4. **Authentication & Authorization**:
   - Handles secure JWT or secure cookie-based session management.
   - Integrates OAuth2 providers and manages document-level access control lists.

---

## 🛠 Tech Stack

- **Language**: [Go (Golang)](https://go.dev/) (1.26+)
- **Database**: PostgreSQL (Users, Projects, ACL, CRDT metadata)
- **Object Storage**: S3 / MinIO (Binary assets, images, compiled PDF/SVG artifacts)
- **Web Framework / Router**: [go-chi/chi](https://github.com/go-chi/chi) (v5)
- **CRDT Support**: Yjs-compatible storage and update merging using [ygo](https://github.com/reearth/ygo)
- **Authentication**: JWT (JSON Web Tokens) using [golang-jwt/jwt](https://github.com/golang-jwt/jwt) (v5) and password hashing with [crypto/bcrypt](https://golang.org/x/crypto/bcrypt)
- **Configuration**: JSON-based configs (`configs/config.json`)

---

## 🗺 Roadmap

- [ ] **Phase 1: Persistence & Storage Layer**
  - [ ] **PostgreSQL**: Replace in-memory repository with PostgreSQL (`pgx` / `sqlx`) for users, projects, ACL, and file metadata.
  - [ ] **S3 / MinIO Storage**: Integrate S3-compatible object storage for binary assets (images, fonts, PDFs).

- [ ] **Phase 2: Real-time Collaboration (WebSockets)**
  - [ ] **WebSocket Gateway**: Implement persistent WebSocket endpoints (`/ws/projects/{projectID}`) for real-time Yjs CRDT delta broadcasting.
  - [ ] **Presence & Cursors**: Real-time position tracking for user cursors, text selections, and active collaborator status.

- [ ] **Phase 3: Version History & Document Snapshots**
  - [ ] **Document Snapshots**: Automated and manual snapshot creation for document state checkpoints.
  - [ ] **Version History Tree**: API for viewing version history, comparing diffs, and restoring previous snapshots.

- [ ] **Phase 4: Monitoring, Logging & Observability**
  - [ ] **Structured Logging**: Migrate logging to `log/slog` / `zap` with contextual trace IDs.
  - [ ] **Prometheus Metrics**: Expose `/metrics` endpoint for tracking active WebSocket connections, CRDT merge latencies, and HTTP throughput.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
