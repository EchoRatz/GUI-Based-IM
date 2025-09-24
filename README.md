<div id="top">

<!-- HEADER STYLE: CLASSIC -->
<div align="left">


# GUI-BASED-IM

<em>Connect Instantly, Communicate Seamlessly, Lead Boldly</em>

<!-- BADGES -->
<img src="https://img.shields.io/github/last-commit/EchoRatz/GUI-Based-IM?style=flat&logo=git&logoColor=white&color=0080ff" alt="last-commit">
<img src="https://img.shields.io/github/languages/top/EchoRatz/GUI-Based-IM?style=flat&color=0080ff" alt="repo-top-language">
<img src="https://img.shields.io/github/languages/count/EchoRatz/GUI-Based-IM?style=flat&color=0080ff" alt="repo-language-count">

<em>Built with the tools and technologies:</em>

<img src="https://img.shields.io/badge/JSON-000000.svg?style=flat&logo=JSON&logoColor=white" alt="JSON">
<img src="https://img.shields.io/badge/Markdown-000000.svg?style=flat&logo=Markdown&logoColor=white" alt="Markdown">
<img src="https://img.shields.io/badge/npm-CB3837.svg?style=flat&logo=npm&logoColor=white" alt="npm">
<img src="https://img.shields.io/badge/.ENV-ECD53F.svg?style=flat&logo=dotenv&logoColor=black" alt=".ENV">
<img src="https://img.shields.io/badge/JavaScript-F7DF1E.svg?style=flat&logo=JavaScript&logoColor=black" alt="JavaScript">
<br>
<img src="https://img.shields.io/badge/MongoDB-47A248.svg?style=flat&logo=MongoDB&logoColor=white" alt="MongoDB">
<img src="https://img.shields.io/badge/Go-00ADD8.svg?style=flat&logo=Go&logoColor=white" alt="Go">
<img src="https://img.shields.io/badge/Gin-008ECF.svg?style=flat&logo=Gin&logoColor=white" alt="Gin">
<img src="https://img.shields.io/badge/Docker-2496ED.svg?style=flat&logo=Docker&logoColor=white" alt="Docker">

</div>
<br>

---

## 📄 Table of Contents

- [Overview](#-overview)
- [Project Structure](#-project-structure)

---

## ✨ Overview

GUI-Based-IM is a modern, real-time messaging platform designed for seamless communication and scalable deployment. Built with a Go backend, MongoDB, and WebSocket support, it enables developers to create robust chat applications with ease.

**Why GUI-Based-IM?**

This project simplifies building real-time chat systems with a focus on performance, security, and ease of deployment. The core features include:

- 🧩 **🚀 WebSocket Support:** Facilitates instant, bidirectional communication for real-time messaging.
- 🛠️ **🔒 User Authentication:** Implements secure, passwordless sign-in and JWT-based access control.
- 🐳 **🖥️ Docker Integration:** Streamlines deployment with Docker Compose and Dockerfiles for consistent environments.
- 📦 **🗃️ Data Persistence:** Uses MongoDB for reliable storage of messages, conversations, and user data.
- ⚙️ **🔧 Easy Setup:** Includes seed and reset scripts for rapid development and testing.

---

## 📁 Project Structure

```sh
└── GUI-Based-IM/
    ├── README.md
    ├── backend
    │   ├── .DS_Store
    │   ├── Dockerfile
    │   ├── auth.go
    │   ├── conver.go
    │   ├── db.go
    │   ├── go.mod
    │   ├── go.sum
    │   ├── main.go
    │   ├── messages.go
    │   ├── receipts.go
    │   └── websocket.go
    ├── docker-compose.yml
    ├── frontend
    │   └── index.html
    ├── package.json
    └── scripts
        ├── reset.mjs
        └── seed.mjs
```

---

<div align="left"><a href="#top">⬆ Return</a></div>

---
