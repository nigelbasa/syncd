# Syncd

A lightweight, high-performance Change Data Capture (CDC) and two-way synchronization service written in Go. 

Syncd is designed for offline-first local applications that need to seamlessly bi-directionally sync their local lightweight databases (like SQLite) with their remote primary databases (like PostgreSQL), all without requiring changes to the existing host applications' logic or controllers.

## Features
- **Trigger-Based CDC**: Changes are detected at the database level, ensuring 100% fidelity without application-level logic.
- **Out-of-Band Sync Logs**: Sync events can be written to a completely separate database file/schema, keeping your primary application database clean.
- **Infinite Loop Protection**: Safely replays remote changes locally (and vice versa) without triggering an endless echo loop of sync events.
- **Database Agnostic (Future)**: Currently supports SQLite <-> PostgreSQL, with future support planned for MongoDB.
- **Standalone Daemon**: Runs as an independent binary service alongside your Electron/Desktop app or backend server.

## Getting Started

*(Documentation to be populated as the Go application is developed)*
