# Go User Management System

Enterprise-ready **User Management & Authentication System** built from scratch using **Golang**, **Gin**, **MongoDB**, and **React (Vite)**.

This project focuses on **clean architecture**, **scalability**, and **real-world authentication workflows**, implemented step by step for deep learning and practical understanding.

---

## 🚀 Features

### 🔐 Authentication
- User Registration
- User Login
- Two-Step Verification (2FA)
- Logout
- Forgot Password
- Reset Password
- Change Password

### 👥 Team & Invite Management
- Create Teams
- Invite Team Members
- Accept / Verify Invites
- Resend Invite
- Activate / Deactivate Members
- Remove Members
- Role-based access control (Admin / Member)

### 🆔 Identity Strategy
- UUID as primary public identifier
- MongoDB ObjectID supported as an alternative (internal usage)
- Secure token-based authentication (JWT)

### 📧 Email (SMTP)
- Email verification
- Password reset emails
- Team invitation emails
- OTP delivery for 2FA

---

## 🏗️ Tech Stack

### Backend
- **Go (Golang)**
- **Gin** – HTTP web framework
- **Gorilla Mux** – Advanced routing concepts
- **MongoDB**
- **JWT** – Token-based authentication
- **SMTP** – Email delivery

### Frontend (Planned)
- **React**
- **Vite**

---

## 📁 Project Structure

```text
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
│   ├── middleware/     # Auth & error handling
│   └── utils/          # Validators, password helpers
│
├── providers/          # External service adapters
│   ├── mongo/          # MongoDB provider
│   ├── smtp/           # Email provider
│   └── token/          # JWT, UUID, OTP handling
│
├── pkg/                # Reusable libraries
│   ├── logger/         # Logging utility
│   └── mongodb/        # MongoDB wrapper
│
├── routes/             # API route definitions
│
└── docs/               # API documentation (future)
```

### 🔒 Security Practices

- Password hashing (bcrypt or equivalent)
- Token expiration and refresh strategy
- Invite and reset tokens with expiry
- Email verification before account activation
- Session invalidation on password reset

### 🎯 Learning Goals

- This project is designed to:
- Understand real-world authentication systems
- Practice clean architecture in Go
- Learn enterprise-level project structuring
- Build confidence designing backend systems from scratch

### 🚀 Future Enhancements

- Role-based permissions
- Audit logs
- Rate limiting
- Account lockout protection
- OAuth (Google / GitHub)
- Background workers for email processing
- Admin dashboard
