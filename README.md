# Last War: Survival - Alliance Manager

A comprehensive Go web application for managing your alliance in the online game Last War: Survival. Track members, manage train schedules, award achievements, and generate communication messages.

## Screenshots

| Members | Train Schedule |
|---|---|
| ![Members](docs/screenshots/members.png) | ![Train Schedule](docs/screenshots/train-schedule.png) |

| Rankings | Member Detail |
|---|---|
| ![Rankings](docs/screenshots/rankings.png) | ![Member Detail](docs/screenshots/member-detail.png) |

## Features

### Core Management
- **Authentication System**: Secure login/logout with session management and password management
- **Role-Based Permissions**: Different access levels for Admin, R5, R4, and lower ranks
- **User-Member Linking**: Users are linked to alliance members with role inheritance
- **Member Management**: Add, edit, delete alliance members, and create user accounts
- **Rank System**: Pre-configured with 5 ranks (R5, R4, R3, R2, R1)

### Train Schedule System
- **Weekly Schedule Management**: Organize and track train conductors and backups
- **Auto-Schedule**: Automatically assign conductors for the week based on performance rankings
- **Performance Tracking**: Track conductor scores and show-up history
- **Weekly Message Generator**: Create formatted messages for alliance chat with schedules
- **Daily Message Generator**: Generate daily reminders for conductors and backups with specific times (15:00 ST/17:00 UK for conductor, 16:30 ST/18:30 UK for backup)

### Awards & Recommendations
- **Weekly Awards**: Track 1st, 2nd, and 3rd place winners across multiple categories
- **Recommendations**: Member recommendation system to boost rankings
- **Performance Rankings**: Real-time leaderboard with detailed score breakdown

### Ranking & Auto-Schedule System
- **Configurable Point System**: Customize points for awards, recommendations, and penalties
- **Smart Conductor Selection**: Automatically selects top 7 performers as conductors
- **Fair Distribution**: Penalties for recent conductors and above-average usage
- **Rank Boosts**: Special bonuses for R4/R5 members and first-time conductors
- **Backup System**: Smart backup assignment from R4/R5 members not in conductor pool

### Communication Tools
- **Customizable Templates**: Configure weekly and daily message templates
- **Train-Themed Messaging**: Fun, themed messages using train lingo ("ALL ABOARD", "Conductor", "Backup Engineer")
- **Placeholder System**: Dynamic message generation with member names, ranks, dates, and times
- **Copy-to-Clipboard**: Easy copying of generated messages for in-game chat

### Screenshot Upload with Image Recognition
- **OCR Processing**: Upload game screenshots for automatic data extraction using Tesseract OCR
- **Intelligent Image Preprocessing**: AI-powered region detection and enhancement
  - Automatically detects and crops data regions (removes headers, tabs, buttons)
  - Enhances contrast and applies scaling for better text recognition
  - Filters out UI elements to focus only on relevant data
- **Smart Parsing**: Advanced pattern matching for names and numeric values
- **Fuzzy Member Matching**: Automatically matches OCR text to database members
- **Manual Entry**: Alternative text-based input for manual data entry
- **Power History Tracking**: Track member power progression over time
- **Mobile-Friendly Interface**: Dedicated upload page optimized for mobile devices

See [IMAGE_RECOGNITION.md](IMAGE_RECOGNITION.md) for detailed technical documentation on the image analysis system.

### Additional Features
- **Profile Management**: Users can change passwords and view account information
- **Settings Page**: R5/Admin-only configuration for ranking system and message templates
- **Responsive UI**: Clean, modern interface that works on desktop and mobile
- **Real-time Filtering**: Filter rankings and schedules by name and rank
- **SQLite Database**: Lightweight, file-based storage for easy deployment

## Prerequisites

### Docker / Podman (Recommended)
The easiest way to run the application — no Go, GCC, or Tesseract installation required.

- **Docker** or **Podman** (with Compose support)

### Building from Source
- **Go 1.23 or higher** — https://golang.org/dl/
- **GCC + G++** (for CGO/Tesseract):
  - Windows: MinGW-w64 or TDM-GCC
  - Linux: `sudo apt-get install build-essential`
  - macOS: Xcode Command Line Tools
- **Tesseract OCR**:
  - Windows: https://github.com/UB-Mannheim/tesseract/wiki
  - Linux: `sudo apt-get install tesseract-ocr tesseract-ocr-all libtesseract-dev libleptonica-dev`
  - macOS: `brew install tesseract`

**Note**: CGO must be enabled (`go env CGO_ENABLED` → `1`). Deploy to Linux for full OCR functionality.

### Production (Debian/Ubuntu Server)
See [DEPLOYMENT.md](DEPLOYMENT.md) for the comprehensive production guide covering:
- Docker Compose deployment (recommended)
- Automated bare-metal installation script
- Let's Encrypt SSL (Caddy or Nginx)
- Security hardening, systemd, firewall, fail2ban, and automated backups

## Installation

### Docker / Podman (Quickest)

Production images are published to GitHub Container Registry — no clone or Go toolchain needed:

```bash
# Generate a session key once:
export SESSION_KEY=$(openssl rand -hex 32)

docker compose up -d          # Docker
# -- or --
podman-compose up -d          # Podman

# The app is now running at http://localhost:8080
# The SQLite database is persisted in ./data/alliance.db
```

To test local code changes instead, build from source with the dev compose file:
`docker compose -f docker-compose.dev.yml up -d --build`

See [DEPLOYMENT.md](DEPLOYMENT.md) for full Docker production setup (with HTTPS reverse proxy).

### Build from Source

1. Navigate to the project directory and download dependencies:

```bash
go mod download
```

2. Build and run:

```bash
go build -o alliance-manager main.go
./alliance-manager
```

### Bare-Metal Production (Debian/Ubuntu)

```bash
chmod +x install.sh
sudo ./install.sh
```

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed manual setup instructions.

## Running the Application

### Docker / Podman

```bash
# Start (detached)
docker compose up -d

# View logs
docker compose logs -f

# Stop
docker compose down

# Rebuild from local source after code changes (dev compose)
docker compose -f docker-compose.dev.yml up -d --build
```

The database is stored in `./data/alliance.db` on the host.

### Development (from source)

Build and run the server:
```bash
go run main.go
```

Or build an executable:
```bash
go build -o alliance-manager
./alliance-manager
```

The application will be available at `http://localhost:8080`.

### Production (bare-metal)

```bash
# Set environment variables
export SESSION_KEY=$(openssl rand -hex 32)
export DATABASE_PATH=/var/lib/lastwar/alliance.db

# Build and run
go build -o alliance-manager main.go
./alliance-manager
```

Or use the systemd service (see [DEPLOYMENT.md](DEPLOYMENT.md)).

## Local Testing

The test suite is built with [Playwright](https://playwright.dev/) and runs against a live local instance. The fastest way to get a working local instance is Docker.

### 1. Start the App

```bash
docker compose -f docker-compose.dev.yml up -d --build
```

The app is now at `http://localhost:8080`. Default credentials: `admin` / `admin123`.

### 2. Install Playwright

```bash
cd playwright-tests
npm install
npx playwright install chromium
```

### 3. Run the Tests

```bash
# All tests (headless)
npx playwright test
# or
npm test

# Single spec
npx playwright test tests/app.spec.js

# Show browser window while running
npx playwright test --headed
```

### 4. View the HTML Report

```bash
npm run report
# opens playwright-tests/report/index.html in a local server
```

### Test Files

| File | What it covers |
|---|---|
| `tests/app.spec.js` | Login flow, member CRUD, train schedule, awards, settings, navigation |
| `tests/vs-ocr.spec.js` | VS Points OCR upload — requires a local screenshot file (see comments in the file) |

> **Tip:** `tests/vs-ocr.spec.js` automatically skips if the required screenshot is not present, so it will not block CI or general test runs.

### Rebuilding After Code Changes

```bash
docker compose -f docker-compose.dev.yml up -d --build   # rebuild image and restart container
npx playwright test                                      # re-run tests against the updated container
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DATABASE_PATH` | `./alliance.db` | Path to SQLite database file |
| `SESSION_KEY` | *(auto-generated)* | 64-character hex string for session encryption. Set a stable value in production or sessions reset on restart. |
| `PRODUCTION` | — | Set to `true` to enable secure cookies (requires HTTPS) |
| `HTTPS` | — | Set to `true` when behind HTTPS to set the Secure cookie flag |

## Default Login Credentials

- **Username**: `admin`
- **Password**: `admin123`

⚠️ **Important**: Change the default password immediately after first login!

## Project Structure

```
LastWar/
├── main.go                  # Go server and all API routes
├── go.mod / go.sum          # Go module dependencies
├── Dockerfile               # Multi-stage container build
├── docker-compose.yml       # Production compose (pulls GHCR image)
├── docker-compose.dev.yml   # Dev compose (builds from local source)
├── .env.example             # Environment variable reference
├── install.sh               # Automated Debian/Ubuntu installation
├── lastwar.service          # Systemd service unit file
├── Caddyfile                # Caddy reverse-proxy configuration
├── data/                    # SQLite database volume (Docker/local)
│   └── alliance.db          # Created automatically on first run
├── DEPLOYMENT.md            # Full production deployment guide
├── QUICKSTART.md            # Fast-path setup reference
├── IMAGE_RECOGNITION.md     # OCR system technical documentation
└── static/                  # Frontend (HTML / CSS / JS)
    ├── index.html           # Member management
    ├── login.html           # Login page
    ├── profile.html         # User profile & password management
    ├── train.html           # Train schedule management
    ├── awards.html          # Awards tracking
    ├── recommendations.html # Recommendation system
    ├── rankings.html        # Performance rankings
    ├── settings.html        # Configuration (R5/Admin only)
    ├── storm.html           # Storm assignments
    ├── vs.html              # VS points tracking
    ├── vs-compliance.html   # VS compliance report
    ├── upload.html          # Screenshot upload (OCR)
    ├── graveyard.html       # Deleted members archive
    ├── conduct.html         # Conduct reports
    ├── admin.html           # Admin panel
    ├── styles.css           # Global styles
    └── *.js                 # Page-specific JavaScript modules
```

## Ranks

- **R5** - Highest rank - Can manage all members
- **R4** - Second highest rank - Can manage all members
- **R3** - Mid-level rank - View only
- **R2** - Lower rank - View only
- **R1** - Lowest rank - View only

## Permissions

- **Admin**: Full access to all features, R5/Admin-only settings (not linked to a member)
- **R5 Members**: Can manage members, create user accounts, update settings, manage all schedules, upload screenshots
- **R4 Members**: Can manage members and schedules, upload screenshots (cannot update settings or create users)
- **R3 Members**: Can upload screenshots and view all information (cannot modify members or schedules)
- **R2/R1 Members**: Can view all information but cannot modify or upload

### R5/Admin-Only Features
- Create user accounts for members
- Update ranking system configuration
- Modify message templates
- Change all system settings

### Upload Features (R3+)
- Upload power ranking screenshots
- Upload VS Points screenshots  
- Manual data entry for power/VS points

## Technologies Used

- **Backend**: Go 1.23, Gorilla Mux, Gorilla Sessions
- **Database**: SQLite (modernc.org/sqlite — pure Go, no CGO needed for DB)
- **OCR**: Tesseract via gosseract (CGO — requires GCC/G++ and Tesseract at build time)
- **Frontend**: Vanilla HTML / CSS / JavaScript
- **Container**: Docker / Podman (multi-stage Alpine build)

## API Endpoints

### Authentication
- `POST /api/login` - User login
- `POST /api/logout` - User logout
- `GET /api/check-auth` - Check authentication status
- `POST /api/change-password` - Change user password

### Member Management (Protected)
- `GET /api/members` - Get all members
- `POST /api/members` - Create a new member (R4/R5 only)
- `PUT /api/members/{id}` - Update a member (R4/R5 only)
- `DELETE /api/members/{id}` - Delete a member (R4/R5 only)
- `POST /api/members/{id}/create-user` - Create user account for member (R5/Admin only)

### Train Schedule (Protected)
- `GET /api/train-schedules` - Get all schedules
- `POST /api/train-schedules` - Create schedule entry
- `PUT /api/train-schedules/{id}` - Update schedule
- `DELETE /api/train-schedules/{id}` - Delete schedule
- `POST /api/train-schedules/auto-schedule` - Auto-assign week's conductors
- `GET /api/train-schedules/weekly-message` - Generate weekly message
- `GET /api/train-schedules/daily-message` - Generate daily conductor message

### Awards (Protected)
- `GET /api/awards` - Get all awards
- `POST /api/awards` - Save awards for a week

### Recommendations (Protected)
- `GET /api/recommendations` - Get all recommendations
- `POST /api/recommendations` - Add recommendation
- `DELETE /api/recommendations/{id}` - Remove recommendation

### Rankings (Protected)
- `GET /api/rankings` - Get member performance rankings

### Settings (R5/Admin Only)
- `GET /api/settings` - Get current settings
- `PUT /api/settings` - Update settings

## Notes

- The database is created automatically on first run (default: `./alliance.db`; Docker: `./data/alliance.db` on the host)
- Port `8080` is used by default (not configurable via env — change in `main.go` if needed)
- All data is stored locally in the SQLite database
- A default `admin` user is created on first run — **change the password immediately**
- Passwords are hashed with bcrypt
- Sessions use signed cookies via Gorilla Sessions; set `SESSION_KEY` to a stable hex value in production
- The Docker image includes Tesseract and English language data — no host-side OCR setup needed
- Set `DATABASE_PATH` to point to the mounted volume path when running in Docker

## How Auto-Schedule Works

The auto-schedule system calculates scores for each member based on:

1. **Award Points**: Points from last week's 1st/2nd/3rd place awards
2. **Recommendation Points**: Points per active recommendation
3. **R4/R5 Rank Boost**: Bonus points for R4 and R5 members
4. **First-Time Conductor Boost**: Extra points for members who've never been conductor
5. **Recent Conductor Penalty**: Reduced points if they were conductor recently
6. **Above Average Penalty**: Penalty for members who've been conductor more than average

The top 7 members are selected as conductors for the week. Backups are selected from R4/R5 members who are not conductors, with each backup used only once per week.

## Message Templates

### Weekly Message Placeholders
- `{WEEK}` - Week start date
- `{SCHEDULES}` - Daily conductor/backup list
- `{NEXT_3}` - Next 3 top-ranked candidates

### Daily Message Placeholders
- `{DATE}` - Formatted date (e.g., Monday, Jan 2, 2006)
- `{CONDUCTOR_NAME}` - Name of the conductor
- `{CONDUCTOR_RANK}` - Rank of the conductor
- `{BACKUP_NAME}` - Name of the backup
- `{BACKUP_RANK}` - Rank of the backup

## Security

- Passwords are hashed with bcrypt before storage
- Session-based authentication with secure cookies
- Role-based access control for all sensitive operations
- SQL injection prevention through parameterized queries
- R5/Admin-only restrictions on critical settings
