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

- **Language**: [Go (Golang)](https://go.dev/) (1.22+)
- **Database**: ???
- **Web Framework / Router**: Lightweight router Chi
- **CRDT Support**: Yjs-compatible update storage and merging logic in Go

---

## 🗺 Roadmap

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
