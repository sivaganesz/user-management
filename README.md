# User Management 
Go User Management System

An enterprise-ready User Management & Authentication System built from scratch using Golang, Gin, MongoDB, and React (Vite).
This project focuses on clean architecture, scalability, and real-world authentication flows.

🚧 Status: In active development
🎯 Goal: Learn and implement a production-grade auth system step by step

✨ Features
🔐 Authentication

User Registration

User Login

Two-Step Verification (2FA)

Logout

Forgot Password

Reset Password

Change Password

👥 Team & Invite Management

Create Teams

Invite Team Members

Accept / Verify Invites

Resend Invite

Activate / Deactivate Members

Remove Members

Role-based access (Admin / Member)

🆔 Identity Strategy

UUID as primary public identifier

MongoDB ObjectID supported as alternative (internal usage)

Secure token-based authentication (JWT)

📧 Email (SMTP)

Email verification

Password reset emails

Team invitation emails

OTP delivery (for 2FA)

🏗️ Tech Stack
Backend

Go (Golang)

Gin – HTTP web framework

Gorilla Mux – Advanced routing concepts

MongoDB

JWT – Token-based authentication

SMTP – Email delivery

Frontend (Planned) Not-started

React

Vite

📁 Project Structure (Enterprise-Oriented)
backend/
├── cmd/                # Application entry points
│   └── api/            # API server
│
├── config/             # Configuration & environment setup
│
├── internal/           # Private application logic
│   ├── controllers/    # Request handlers
│   ├── services/       # Business logic
│   ├── models/         # Database models
│   ├── middleware/     # Auth, error handling, etc.
│   └── utils/          # Validators, password helpers
│
├── providers/          # External service adapters
│   ├── mongo/          # MongoDB provider
│   ├── smtp/           # Email provider
│   └── token/          # JWT, UUID, OTP
│
├── pkg/                # Reusable libraries
│   ├── logger/         # Logging utility
│   └── mongodb/        # MongoDB wrapper
│
├── routes/             # API route definitions
│
└── docs/               # API documentation (future)

🔄 Request Flow (High-Level)
Client (React)
   ↓
Routes
   ↓
Controllers
   ↓
Services
   ↓
Providers (DB / SMTP / Tokens)
   ↓
MongoDB / Email


Each layer has a single responsibility, making the system easy to test, debug, and scale.

🔒 Security Practices

Password hashing (bcrypt or equivalent)

Token expiration & refresh strategy

Invite & reset tokens with expiry

Email verification before activation

Session invalidation on password reset

🎯 Learning Goals

This project is built to:

Understand real-world authentication flows

Practice clean architecture in Go

Learn enterprise-level project structuring

Gain confidence building systems from scratch

🚀 Future Enhancements

Role-based permissions

Audit logs

Rate limiting

Account lockout protection

OAuth (Google/GitHub)

Admin dashboard

Background workers for email jobs

🤝 Contributions

This project is primarily for learning, but suggestions and discussions are welcome.