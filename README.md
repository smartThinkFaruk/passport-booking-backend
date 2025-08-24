# Folder PATH listing for Go Lang Application
### Volume serial number is 7235-95A8
```text
│   .env
│   .gitignore
│   go.mod
│   go.sum
│   main.go
│   README.md
│
├───controllers
│       auth.go
│       counter.go
│       user.go
│
├───database
│   │   database.go
│   │
│   └───migrations
│           migrations.go
│           registry.go
│
├───log
│   └───app
│           app_06-04-2025.log
│
├───logger
│       logger.go
│
├───middleware
│       jwt.go
│
├───models
│       user.go
│
├───resource
│       resource.go
│
├───routes
│       routes.go
│
├───types
│       login.go
│       register.go
│       response.go
│
└───utils
        utils.go
```

# Run This Application
```text
go run .
```
## ✅ Why Use a .env File in Go Projects?

#### The .env file is used to store environment variables — configuration values that can change between environments (local, staging, production). This helps you:
<ul>
<li>Avoid hardcoding sensitive data like DB passwords or secret keys in your code.</li>
<li>Easily switch environments without changing code.</li>
<li>Improve security by keeping secrets out of version control (usually added to .gitignore).</li>
</ul>
You typically load this file using a library like github.com/joho/godotenv.

## 🔍 Breakdown of Your .env Variables
### 🔧 App Configurations
```text
APP_NAME=TR-Tech-Course            # Name of your app (for logs or UI use)
APP_ENV=local                      # Current environment (e.g., local, production)
APP_DEBUG=true                     # Enables debug logs if true
APP_TIMEZONE=UTC                   # App's timezone (for time-related functions)
APP_PORT=8081                      # Port your app will run on
APP_HOST=localhost                 # Host where app is running
```
### 🛢️ Database Configurations
```text
DB_CONNECTION=mysql                # Type of DB (you may use this to choose a driver)
DB_HOST=127.0.0.1                  # Database host (localhost)
DB_PORT=3306                       # Default MySQL port
DB_DATABASE=tr_tech_course_go      # Name of your DB
DB_USERNAME=root                   # DB username
DB_PASSWORD=                       # DB password (keep this secret in prod)
```
### 🔐 Security Config
```text
SECRET_KEY='zbqxMxOci0OTeSo8StJyLLRfTmz3A3Vr4b4R6Fp2rtUMLqmVD6bgyH466xw3D0jz97iqgj5aVkx6IDK04vS3zOWSs3CgOhU2ISXD'
# Used for signing JWT tokens or encryption
```
## 📦 go.mod — The Module Definition File
go.mod is the core configuration file for any Go module. It tells Go:
<ul>
    <li>The name of your module/project (module passport-booking).</li>
    <li>The Go version your project uses (go 1.23.5).</li>
    <li>All required dependencies (require (...)) and their versions.</li>
</ul>

### 🔍 Why you need it:

<ul>
    <li>To enable dependency management.</li>
    <li>To lock the version of every library you use.</li>
    <li>To make the project reproducible (anyone can go build or go run and get the same results).</li>
    <li>To initialize Go modules (introduced in Go 1.11+).</li>
</ul>

### ✅ Example from your go.mod:
```text
require (
github.com/gofiber/fiber/v2 v2.52.6         // Web framework
github.com/golang-jwt/jwt/v5 v5.2.1         // JWT handling
github.com/google/uuid v1.6.0               // UUID generator
github.com/jinzhu/now v1.1.5                // Date/time utilities
github.com/joho/godotenv v1.5.1             // .env loader
golang.org/x/crypto v0.35.0                 // Crypto library
gorm.io/driver/mysql v1.5.7                 // GORM MySQL driver
gorm.io/gorm v1.25.12                       // GORM ORM
)
```

## 🔐 go.sum — The Checksum File

go.sum stores cryptographic checksums (hashes) for every version of every dependency in your project.

### 🔍 Why you need it:
<ul>
    <li>Ensures security: verifies downloaded modules haven’t been tampered with.</li>
    <li>Ensures consistency: same code everywhere (your machine, CI/CD, production).</li>
    <li>Helps Go know what to download and trust.</li>
</ul>

### ✅ Example:
```text
github.com/joho/godotenv v1.5.1 h1:7eLL/+HRGLY0ldzfGMeQkb7vMd0as4CfYvUVzLqw0N0=
github.com/joho/godotenv v1.5.1/go.mod h1:f4LDr5Voq0i2e/R5DDNOoa2zzDfwtkZa6DnEwAbqwq4=
```
This means: for godotenv version v1.5.1, Go has a hash of the module and the module’s go.mod file to verify authenticity.

### ⚙️ How It All Works Together
<ul>
    <li>You write: import "github.com/joho/godotenv"</li>
    <li>Go fetches it → logs it in go.mod with the version.</li>
    <li>Go records the hash → stores it in go.sum.</li>
    <li>Anyone cloning your code and running go mod tidy or go build will get the exact same versions.</li>
</ul>

### 🎯 Summary File	Purpose
<ul>
    <li>go.mod	Declares project name, Go version, and required dependencies.</li>
    <li>go.sum	Ensures integrity and consistency of downloaded dependencies.</li>
</ul>

## SMS Integration

This application integrates with an external SMS API for OTP delivery and notifications. The SMS service is used for:

### Features
- **OTP Delivery**: Sends OTP codes to delivery phone numbers for verification
- **Delivery Notifications**: Sends confirmation messages after successful phone verification
- **Configurable SMS Service**: Supports custom SMS API endpoints and authentication

### Configuration
Add the following environment variables to your `.env` file:

```env
# SMS API Configuration
SMS_API_URL=https://ekdak.com/message-broker/send-sms/
SMS_AUTH_TOKEN=Token 8d3690ef76134d9abd78f9cbde655dd46446a032
```

### API Endpoints

#### Update Delivery Phone
```http
PUT /api/booking/delivery-phone
```
**Request Body:**
```json
{
    "booking_id": 1,
    "delivery_phone": "+8801234567890"
}
```
**Response:** Updates the delivery phone and sends an OTP for verification.

#### Verify Delivery Phone
```http
POST /api/booking/verify-delivery-phone
```
**Request Body:**
```json
{
    "booking_id": 1,
    "phone": "+8801234567890",
    "otp_code": "123456"
}
```
**Response:** Verifies the OTP and marks the delivery phone as verified.

#### Resend OTP
```http
POST /api/booking/resend-otp
```
**Request Body:**
```json
{
    "booking_id": 1,
    "phone": "+8801234567890"
}
```
**Response:** Resends the OTP to the delivery phone.

#### Test SMS (Development Only)
```http
POST /api/booking/test-sms
```
**Request Body:**
```json
{
    "phone": "+8801234567890",
    "message": "Test message"
}
```
**Response:** Sends a test SMS message.

### SMS Message Templates

#### OTP Message
```
Your OTP code is: {OTP_CODE}. This code will expire in 5 minutes. Please do not share this code with anyone.
```

#### Delivery Notification
```
Your passport delivery is confirmed for booking ID: {BOOKING_ID}. Our delivery partner will contact you soon.
```

### Error Handling
- Failed SMS delivery is logged but doesn't prevent OTP creation
- Fallback mechanism displays OTP in console for testing when SMS fails
- Configurable retry limits and blocking mechanisms for security

### Security Features
- OTP expiration (5 minutes)
- Retry limit (3 attempts)
- Temporary blocking after failed attempts
- Encrypted OTP storage in database
