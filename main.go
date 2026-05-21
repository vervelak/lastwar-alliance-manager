// Copyright (c) 2026 Vervelak
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	gosseract "github.com/otiai10/gosseract/v2"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type Member struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Nickname       *string `json:"nickname,omitempty"`
	Rank           string  `json:"rank"`
	Eligible       bool    `json:"eligible"`
	Power          *int64  `json:"power,omitempty"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
	DeletionReason *string `json:"deletion_reason,omitempty"`
	DeletedBy      *string `json:"deleted_by,omitempty"`
}

type MemberStats struct {
	ID                   int     `json:"id"`
	Name                 string  `json:"name"`
	Rank                 string  `json:"rank"`
	ConductorCount       int     `json:"conductor_count"`
	LastConductorDate    *string `json:"last_conductor_date"`
	BackupCount          int     `json:"backup_count"`
	BackupUsedCount      int     `json:"backup_used_count"`
	ActualConductorCount int     `json:"actual_conductor_count"`
	ConductorNoShowCount int     `json:"conductor_no_show_count"`
}

type User struct {
	ID       int
	Username string
	Password string
	MemberID *int
	IsAdmin  bool
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TrainSchedule struct {
	ID                  int     `json:"id"`
	Date                string  `json:"date"`
	ConductorID         int     `json:"conductor_id"`
	ConductorName       string  `json:"conductor_name"`
	ConductorScore      *int    `json:"conductor_score"`
	BackupID            int     `json:"backup_id"`
	BackupName          string  `json:"backup_name"`
	BackupRank          string  `json:"backup_rank"`
	ConductorShowedUp   *bool   `json:"conductor_showed_up"`
	ActualConductorID   *int    `json:"actual_conductor_id"`
	ActualConductorName *string `json:"actual_conductor_name"`
	VipID               *int    `json:"vip_id,omitempty"`
	VipName             *string `json:"vip_name,omitempty"`
	Notes               *string `json:"notes"`
	CreatedAt           string  `json:"created_at"`
}

type Award struct {
	ID             int     `json:"id"`
	WeekDate       string  `json:"week_date"`
	AwardType      string  `json:"award_type"`
	Rank           int     `json:"rank"`
	MemberID       int     `json:"member_id"`
	MemberName     string  `json:"member_name"`
	MemberNickname *string `json:"member_nickname,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

type AwardType struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
}

type Recommendation struct {
	ID              int     `json:"id"`
	MemberID        int     `json:"member_id"`
	MemberName      string  `json:"member_name"`
	MemberRank      string  `json:"member_rank"`
	MemberNickname  *string `json:"member_nickname,omitempty"`
	RecommendedBy   string  `json:"recommended_by"`
	RecommendedByID int     `json:"recommended_by_id"`
	Notes           string  `json:"notes"`
	CreatedAt       string  `json:"created_at"`
	Expired         bool    `json:"expired"`
}

type ConductReport struct {
	ID             int     `json:"id"`
	MemberID       int     `json:"member_id"`
	MemberName     string  `json:"member_name"`
	MemberRank     string  `json:"member_rank"`
	MemberNickname *string `json:"member_nickname,omitempty"`
	Points         int     `json:"points"`
	Notes          string  `json:"notes"`
	CreatedBy      string  `json:"created_by"`
	CreatedByID    int     `json:"created_by_id"`
	CreatedAt      string  `json:"created_at"`
	Expired        bool    `json:"expired"`
}

type WeekAwards struct {
	WeekDate string             `json:"week_date"`
	Awards   map[string][]Award `json:"awards"`
}

type PowerHistory struct {
	ID         int    `json:"id"`
	MemberID   int    `json:"member_id"`
	Power      int64  `json:"power"`
	RecordedAt string `json:"recorded_at"`
}

type LoginSession struct {
	ID        int     `json:"id"`
	UserID    int     `json:"user_id"`
	Username  string  `json:"username"`
	IPAddress *string `json:"ip_address,omitempty"`
	UserAgent *string `json:"user_agent,omitempty"`
	Country   *string `json:"country,omitempty"`
	City      *string `json:"city,omitempty"`
	ISP       *string `json:"isp,omitempty"`
	LoginTime string  `json:"login_time"`
	Success   bool    `json:"success"`
}

type IPGeolocation struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
}

type AdminUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	MemberID *int   `json:"member_id,omitempty"`
	IsAdmin  bool   `json:"is_admin"`
}

type AdminUserResponse struct {
	ID           int            `json:"id"`
	Username     string         `json:"username"`
	MemberID     *int           `json:"member_id,omitempty"`
	MemberName   *string        `json:"member_name,omitempty"`
	IsAdmin      bool           `json:"is_admin"`
	CreatedAt    string         `json:"created_at,omitempty"`
	LastLogin    *string        `json:"last_login,omitempty"`
	LoginCount   int            `json:"login_count"`
	RecentLogins []LoginSession `json:"recent_logins,omitempty"`
}

type Applicant struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Power           *int64  `json:"power,omitempty"`
	Rank            *string `json:"rank,omitempty"`
	VouchedBy       *string `json:"vouched_by,omitempty"`
	Status          string  `json:"status"` // pending, approved, rejected, on_trial
	DecisionBy      *string `json:"decision_by,omitempty"`
	AppliedAt       string  `json:"applied_at"`
	DecidedAt       *string `json:"decided_at,omitempty"`
	TrialEndDate    *string `json:"trial_end_date,omitempty"`
	Notes           *string `json:"notes,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
	MemberID        *int    `json:"member_id,omitempty"`
}

type Settings struct {
	ID                           int    `json:"id"`
	AllianceName                 string `json:"alliance_name"`       // Full alliance name (e.g., "Reset Reapers")
	AllianceShortName            string `json:"alliance_short_name"` // Short alliance tag (e.g., "RSRP")
	AwardFirstPoints             int    `json:"award_first_points"`
	AwardSecondPoints            int    `json:"award_second_points"`
	AwardThirdPoints             int    `json:"award_third_points"`
	RecommendationPoints         int    `json:"recommendation_points"`
	RecentConductorPenaltyDays   int    `json:"recent_conductor_penalty_days"`
	AboveAverageConductorPenalty int    `json:"above_average_conductor_penalty"`
	R4R5RankBoost                int    `json:"r4r5_rank_boost"`
	FirstTimeConductorBoost      int    `json:"first_time_conductor_boost"`
	ScheduleMessageTemplate      string `json:"schedule_message_template"`
	DailyMessageTemplate         string `json:"daily_message_template"`
	PowerTrackingEnabled         bool   `json:"power_tracking_enabled"`
	ServerTimezone               string `json:"server_timezone"`
	ConductorTime                string `json:"conductor_time"`
	BackupTime                   string `json:"backup_time"`
	DisplayTimezones             string `json:"display_timezones"`
	VSPointsDailyTarget          int    `json:"vs_points_daily_target"`
	VSPointsWeeklyTarget         int    `json:"vs_points_weekly_target"`
	MinPower                     int    `json:"min_power"`
	MinHQLevel                   int    `json:"min_hq_level"`
	VipSeatEnabled               bool   `json:"vip_seat_enabled"`
	MarshalGuardEnabled          bool   `json:"marshal_guard_enabled"`
}

type MemberRanking struct {
	Member                  Member        `json:"member"`
	TotalScore              int           `json:"total_score"`
	AwardPoints             int           `json:"award_points"`
	RecommendationPoints    int           `json:"recommendation_points"`
	RecentConductorPenalty  int           `json:"recent_conductor_penalty"`
	AboveAveragePenalty     int           `json:"above_average_penalty"`
	RankBoost               int           `json:"rank_boost"`
	FirstTimeConductorBoost int           `json:"first_time_conductor_boost"`
	ConductorCount          int           `json:"conductor_count"`
	LastConductorDate       *string       `json:"last_conductor_date"`
	DaysSinceLastConductor  *int          `json:"days_since_last_conductor"`
	AwardDetails            []AwardDetail `json:"award_details"`
	RecommendationCount     int           `json:"recommendation_count"`
	MGEventCount            int           `json:"mg_event_count"`
	MGTotalDamage           int64         `json:"mg_total_damage"`
}

type AwardDetail struct {
	AwardType string `json:"award_type"`
	Rank      int    `json:"rank"`
	Points    int    `json:"points"`
	WeekDate  string `json:"week_date"`
	Expired   bool   `json:"expired"`
}

type StormAssignment struct {
	ID         int    `json:"id"`
	TaskForce  string `json:"task_force"`
	BuildingID string `json:"building_id"`
	MemberID   int    `json:"member_id"`
	Position   int    `json:"position"`
}

type DetectedMember struct {
	Name         string   `json:"name"`
	Rank         string   `json:"rank"`
	IsNew        bool     `json:"is_new"`
	RankChanged  bool     `json:"rank_changed"`
	OldRank      string   `json:"old_rank,omitempty"`
	SimilarMatch []string `json:"similar_match,omitempty"`
}

type RenameInfo struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

type MemberToRemove struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Rank string `json:"rank"`
}

type ConfirmRequest struct {
	Members         []DetectedMember `json:"members"`
	RemoveMemberIDs []int            `json:"remove_member_ids"`
	Renames         []RenameInfo     `json:"renames"`
}

type ConfirmResult struct {
	Added     int `json:"added"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Removed   int `json:"removed"`
}

type VSPoints struct {
	ID        int    `json:"id"`
	MemberID  int    `json:"member_id"`
	WeekDate  string `json:"week_date"`
	Monday    int    `json:"monday"`
	Tuesday   int    `json:"tuesday"`
	Wednesday int    `json:"wednesday"`
	Thursday  int    `json:"thursday"`
	Friday    int    `json:"friday"`
	Saturday  int    `json:"saturday"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type VSPointsWithMember struct {
	VSPoints
	MemberName string `json:"member_name"`
	MemberRank string `json:"member_rank"`
}

var db *sql.DB
var store *sessions.CookieStore
var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

// jsonError writes a JSON error response with the given status code and message.
func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// requestLoggingMiddleware logs each HTTP request with method, path, status, and duration.
func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Calculate Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	s1Lower := strings.ToLower(s1)
	s2Lower := strings.ToLower(s2)
	len1 := len(s1Lower)
	len2 := len(s2Lower)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1Lower[i-1] != s2Lower[j-1] {
				cost = 1
			}
			matrix[i][j] = min(matrix[i-1][j]+1, matrix[i][j-1]+1, matrix[i-1][j-1]+cost)
		}
	}
	return matrix[len1][len2]
}

func min(nums ...int) int {
	if len(nums) == 0 {
		return 0
	}
	minNum := nums[0]
	for _, n := range nums[1:] {
		if n < minNum {
			minNum = n
		}
	}
	return minNum
}

// Check if two names are similar (case-insensitive)
func areSimilar(name1, name2 string) bool {
	if strings.EqualFold(name1, name2) {
		return false // Exact match, not similar but same
	}

	// Calculate similarity (case-insensitive)
	lower1 := strings.ToLower(name1)
	lower2 := strings.ToLower(name2)
	dist := levenshteinDistance(lower1, lower2)
	maxLen := max(len(lower1), len(lower2))
	similarity := 1.0 - float64(dist)/float64(maxLen)

	// Consider similar if:
	// 1. Similarity >= 70%
	// 2. Distance <= 3 characters
	// 3. One name contains the other (for abbreviations like IRA vs IRAQ Army)
	if similarity >= 0.7 || dist <= 3 {
		return true
	}

	// Check if one name contains significant part of another
	name1Lower := strings.ToLower(name1)
	name2Lower := strings.ToLower(name2)
	if strings.Contains(name1Lower, name2Lower) || strings.Contains(name2Lower, name1Lower) {
		if len(name1) >= 3 && len(name2) >= 3 {
			return true
		}
	}

	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// loginRateLimiter tracks failed login attempts per IP to prevent brute force attacks.
type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time // IP -> timestamps of recent failures
}

var rateLimiter = &loginRateLimiter{
	attempts: make(map[string][]time.Time),
}

const (
	rateLimitWindow  = 15 * time.Minute
	rateLimitMaxFail = 10 // max failed attempts per IP within the window
)

// isRateLimited returns true if the IP has exceeded the allowed number of failed login attempts.
func (rl *loginRateLimiter) isRateLimited(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	var recent []time.Time
	for _, t := range rl.attempts[ip] {
		if now.Sub(t) < rateLimitWindow {
			recent = append(recent, t)
		}
	}
	rl.attempts[ip] = recent
	return len(recent) >= rateLimitMaxFail
}

// recordFailure records a failed login attempt for the given IP.
func (rl *loginRateLimiter) recordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.attempts[ip] = append(rl.attempts[ip], time.Now())
}

// initSessionStore initializes the session store with secure settings
func initSessionStore() {
	// Get session key from environment or generate a secure one
	sessionKey := os.Getenv("SESSION_KEY")
	isProduction := os.Getenv("PRODUCTION") == "true" || os.Getenv("HTTPS") == "true"

	if sessionKey == "" {
		if isProduction {
			log.Fatal("FATAL: SESSION_KEY environment variable is required in production. Generate one with: openssl rand -hex 32")
		}
		// Generate a random 32-byte key for development
		key := make([]byte, 32)
		rand.Read(key)
		sessionKey = hex.EncodeToString(key)
		log.Println("WARNING: No SESSION_KEY environment variable set. Using generated key (not persistent across restarts).")
		log.Printf("For production, set SESSION_KEY environment variable. Example: export SESSION_KEY=%s", sessionKey)
	} else if isProduction && len(sessionKey) < 32 {
		log.Fatal("FATAL: SESSION_KEY must be at least 32 characters for production. Generate one with: openssl rand -hex 32")
	}

	// Decode hex key
	key, err := hex.DecodeString(sessionKey)
	if err != nil || len(key) != 32 {
		// Fallback: use the string directly if not valid hex
		key = []byte(sessionKey)
		if len(key) < 32 {
			// Pad to 32 bytes
			padded := make([]byte, 32)
			copy(padded, key)
			key = padded
		}
	}

	store = sessions.NewCookieStore(key[:32])

	// Configure secure cookie options
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400,                   // 24 hours
		HttpOnly: true,                    // Prevent JavaScript access
		Secure:   isProduction,            // Only send over HTTPS in production
		SameSite: http.SameSiteStrictMode, // CSRF protection
	}

	if isProduction {
		log.Println("Session cookies configured for HTTPS (Secure flag enabled)")
	} else {
		log.Println("Session cookies configured for HTTP (development mode)")
	}
}

// RankingContext holds all data needed for ranking calculations
type RankingContext struct {
	Settings          Settings
	RecommendationMap map[int]int // memberID -> count
	AwardScoreMap     map[int]int // memberID -> total points
	ConductorStats    map[int]ConductorStat
	AvgConductorCount float64
	ReferenceDate     time.Time
}

type ConductorStat struct {
	Count          int
	LastDate       *string
	LastBackupUsed *string
}

// loadSettings loads the settings from the database
func loadSettings() (Settings, error) {
	var settings Settings
	err := db.QueryRow(`SELECT id, award_first_points, award_second_points, award_third_points, 
		recommendation_points, recent_conductor_penalty_days, above_average_conductor_penalty, r4r5_rank_boost,
		first_time_conductor_boost, schedule_message_template, daily_message_template,
		COALESCE(server_timezone, 'UTC') as server_timezone,
		COALESCE(conductor_time, '15:00') as conductor_time,
		COALESCE(backup_time, '16:30') as backup_time,
		COALESCE(display_timezones, '["Europe/London"]') as display_timezones
		FROM settings WHERE id = 1`).Scan(
		&settings.ID,
		&settings.AwardFirstPoints,
		&settings.AwardSecondPoints,
		&settings.AwardThirdPoints,
		&settings.RecommendationPoints,
		&settings.RecentConductorPenaltyDays,
		&settings.AboveAverageConductorPenalty,
		&settings.R4R5RankBoost,
		&settings.FirstTimeConductorBoost,
		&settings.ScheduleMessageTemplate,
		&settings.DailyMessageTemplate,
		&settings.ServerTimezone,
		&settings.ConductorTime,
		&settings.BackupTime,
		&settings.DisplayTimezones,
	)
	return settings, err
}

// loadRecommendations loads recommendation counts for all members (active only)
// A recommendation is active if the member hasn't been assigned as conductor/backup/actual_conductor after it was created
func loadRecommendations() (map[int]int, error) {
	rows, err := db.Query(`
		SELECT r.member_id, COUNT(*) as rec_count
		FROM recommendations r
		WHERE NOT EXISTS (
			SELECT 1 FROM train_schedules ts
			WHERE (ts.conductor_id = r.member_id 
			       OR (ts.backup_id = r.member_id AND ts.conductor_showed_up = 0 AND ts.actual_conductor_id IS NULL)
			       OR ts.actual_conductor_id = r.member_id)
			AND ts.date >= date(r.created_at)
		)
		GROUP BY r.member_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recommendationMap := make(map[int]int)
	for rows.Next() {
		var memberID, count int
		if err := rows.Scan(&memberID, &count); err != nil {
			return nil, err
		}
		recommendationMap[memberID] = count
	}
	return recommendationMap, nil
}

// loadAwards loads award scores for all members (active only)
// An award is active if the member hasn't been assigned as conductor/backup/actual_conductor after the award week
func loadAwards(settings Settings) (map[int]int, error) {
	rows, err := db.Query(`
		SELECT a.member_id, a.rank
		FROM awards a
		WHERE NOT EXISTS (
			SELECT 1 FROM train_schedules ts
			WHERE (ts.conductor_id = a.member_id 
			       OR (ts.backup_id = a.member_id AND ts.conductor_showed_up = 0 AND ts.actual_conductor_id IS NULL)
			       OR ts.actual_conductor_id = a.member_id)
			AND ts.date >= a.week_date
		)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	awardScoreMap := make(map[int]int)
	for rows.Next() {
		var memberID, rank int
		if err := rows.Scan(&memberID, &rank); err != nil {
			return nil, err
		}
		switch rank {
		case 1:
			awardScoreMap[memberID] += settings.AwardFirstPoints
		case 2:
			awardScoreMap[memberID] += settings.AwardSecondPoints
		case 3:
			awardScoreMap[memberID] += settings.AwardThirdPoints
		}
	}
	return awardScoreMap, nil
}

// loadConductorStats loads conductor statistics for all members
func loadConductorStats() (map[int]ConductorStat, float64, error) {
	rows, err := db.Query(`
		SELECT conductor_id, COUNT(*) as conductor_count, MAX(date) as last_date
		FROM train_schedules
		GROUP BY conductor_id
	`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	conductorStats := make(map[int]ConductorStat)
	totalConductorCount := 0
	memberCount := 0

	for rows.Next() {
		var memberID, count int
		var lastDate sql.NullString
		if err := rows.Scan(&memberID, &count, &lastDate); err != nil {
			return nil, 0, err
		}
		var lastDatePtr *string
		if lastDate.Valid {
			lastDatePtr = &lastDate.String
		}
		conductorStats[memberID] = ConductorStat{
			Count:    count,
			LastDate: lastDatePtr,
		}
		totalConductorCount += count
		memberCount++
	}

	// Load backup usage dates (when conductor didn't show up and backup actually conducted)
	backupRows, err := db.Query(`
		SELECT backup_id, MAX(date) as last_backup_used
		FROM train_schedules
		WHERE conductor_showed_up = 0 AND actual_conductor_id IS NULL
		GROUP BY backup_id
	`)
	if err != nil {
		return nil, 0, err
	}
	defer backupRows.Close()

	for backupRows.Next() {
		var memberID int
		var lastBackupUsed sql.NullString
		if err := backupRows.Scan(&memberID, &lastBackupUsed); err != nil {
			return nil, 0, err
		}
		var lastBackupUsedPtr *string
		if lastBackupUsed.Valid {
			lastBackupUsedPtr = &lastBackupUsed.String
		}

		// Update existing stat or create new one
		if stat, exists := conductorStats[memberID]; exists {
			stat.LastBackupUsed = lastBackupUsedPtr
			conductorStats[memberID] = stat
		} else {
			conductorStats[memberID] = ConductorStat{
				Count:          0,
				LastDate:       nil,
				LastBackupUsed: lastBackupUsedPtr,
			}
		}
	}

	// Load actual conductor dates (when backup assigned the train to another person)
	actualRows, err := db.Query(`
		SELECT actual_conductor_id, COUNT(*) as conductor_count, MAX(date) as last_date
		FROM train_schedules
		WHERE actual_conductor_id IS NOT NULL
		GROUP BY actual_conductor_id
	`)
	if err != nil {
		return nil, 0, err
	}
	defer actualRows.Close()

	for actualRows.Next() {
		var memberID, count int
		var lastDate sql.NullString
		if err := actualRows.Scan(&memberID, &count, &lastDate); err != nil {
			return nil, 0, err
		}
		var lastDatePtr *string
		if lastDate.Valid {
			lastDatePtr = &lastDate.String
		}

		if stat, exists := conductorStats[memberID]; exists {
			stat.Count += count
			if lastDatePtr != nil && (stat.LastDate == nil || *lastDatePtr > *stat.LastDate) {
				stat.LastDate = lastDatePtr
			}
			conductorStats[memberID] = stat
		} else {
			conductorStats[memberID] = ConductorStat{
				Count:    count,
				LastDate: lastDatePtr,
			}
			totalConductorCount += count
			memberCount++
		}
	}

	var avgConductorCount float64
	if memberCount > 0 {
		avgConductorCount = float64(totalConductorCount) / float64(memberCount)
	}

	return conductorStats, avgConductorCount, nil
}

// buildRankingContext creates a complete ranking context for calculations
func buildRankingContext(referenceDate time.Time) (*RankingContext, error) {
	settings, err := loadSettings()
	if err != nil {
		return nil, err
	}

	recommendationMap, err := loadRecommendations()
	if err != nil {
		return nil, err
	}

	// Get all non-expired awards (stacks up over multiple weeks)
	awardScoreMap, err := loadAwards(settings)
	if err != nil {
		return nil, err
	}

	conductorStats, avgConductorCount, err := loadConductorStats()
	if err != nil {
		return nil, err
	}

	return &RankingContext{
		Settings:          settings,
		RecommendationMap: recommendationMap,
		AwardScoreMap:     awardScoreMap,
		ConductorStats:    conductorStats,
		AvgConductorCount: avgConductorCount,
		ReferenceDate:     referenceDate,
	}, nil
}

// calculateMemberScore calculates the ranking score for a member
func calculateMemberScore(member Member, ctx *RankingContext) int {
	score := 0

	// Add recommendation points with non-linear scaling (diminishing returns)
	recCount := ctx.RecommendationMap[member.ID]
	if recCount > 0 {
		// Formula: 5 + 5 * sqrt(recCount) rounded to nearest int
		// This gives: 1 rec = 10pts, 2 recs = 12pts, 3 recs = 14pts, 4 recs = 15pts
		recPoints := 5.0 + 5.0*math.Sqrt(float64(recCount))
		score += int(math.Round(recPoints))
	}

	// Add award points
	score += ctx.AwardScoreMap[member.ID]

	// Add rank boost for R4/R5 members (exponential based on days since last conductor)
	if member.Rank == "R4" || member.Rank == "R5" {
		baseBoost := float64(ctx.Settings.R4R5RankBoost)

		// Calculate days since last conductor/backup duty
		var daysSinceLastDuty int = 0
		if stats, exists := ctx.ConductorStats[member.ID]; exists {
			var mostRecentDate *time.Time

			if stats.LastDate != nil {
				if lastDate, err := parseDate(*stats.LastDate); err == nil {
					mostRecentDate = &lastDate
				}
			}

			// Check if backup usage was more recent
			if stats.LastBackupUsed != nil {
				if backupDate, err := parseDate(*stats.LastBackupUsed); err == nil {
					if mostRecentDate == nil || backupDate.After(*mostRecentDate) {
						mostRecentDate = &backupDate
					}
				}
			}

			if mostRecentDate != nil {
				daysSinceLastDuty = int(ctx.ReferenceDate.Sub(*mostRecentDate).Hours() / 24)
			}
		}

		// Exponential formula: base_boost * 2^(days/7)
		// Doubles every week, guarantees selection within 3 weeks
		multiplier := math.Pow(2, float64(daysSinceLastDuty)/7.0)
		score += int(math.Round(baseBoost * multiplier))
	}

	// Add first time conductor boost if member has never been conductor and has some points
	if stats, exists := ctx.ConductorStats[member.ID]; !exists || stats.Count == 0 {
		// Only give boost if they have some positive score (awards, recommendations, or rank boost)
		if score > 0 {
			score += ctx.Settings.FirstTimeConductorBoost
		}
	}

	// Apply conductor-based penalties
	if stats, exists := ctx.ConductorStats[member.ID]; exists {
		// Penalize if above average conductor count
		if float64(stats.Count) > ctx.AvgConductorCount {
			score -= ctx.Settings.AboveAverageConductorPenalty
		}

		// Penalize recent conductors - check both conductor date and backup used date
		var mostRecentDate *time.Time

		if stats.LastDate != nil {
			if lastDate, err := parseDate(*stats.LastDate); err == nil {
				mostRecentDate = &lastDate
			}
		}

		// If they stepped in as backup, check if that's more recent
		if stats.LastBackupUsed != nil {
			if backupDate, err := parseDate(*stats.LastBackupUsed); err == nil {
				if mostRecentDate == nil || backupDate.After(*mostRecentDate) {
					mostRecentDate = &backupDate
				}
			}
		}

		// Apply penalty based on most recent duty (conductor or backup usage)
		if mostRecentDate != nil {
			daysSince := int(ctx.ReferenceDate.Sub(*mostRecentDate).Hours() / 24)
			penalty := ctx.Settings.RecentConductorPenaltyDays - daysSince
			if penalty > 0 {
				score -= penalty
			}
		}
	}

	return score
}

func initDB() error {
	var err error

	// Use DATABASE_PATH environment variable if set, otherwise use local path
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./alliance.db"
	}

	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// Create members table
	createMembersTableSQL := `CREATE TABLE IF NOT EXISTS members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		rank TEXT NOT NULL,
		eligible BOOLEAN NOT NULL DEFAULT 1
	);`

	_, err = db.Exec(createMembersTableSQL)
	if err != nil {
		return err
	}

	// Migrate existing members table to add eligible column if missing
	var eligibleColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('members')
		WHERE name = 'eligible'
	`).Scan(&eligibleColumnExists)
	if err != nil {
		return err
	}

	if !eligibleColumnExists {
		_, err = db.Exec(`ALTER TABLE members ADD COLUMN eligible BOOLEAN NOT NULL DEFAULT 1`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added eligible column to members table")
	}

	// Migrate members table to add nickname column if missing
	var nicknameColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('members')
		WHERE name = 'nickname'
	`).Scan(&nicknameColumnExists)
	if err != nil {
		return err
	}

	if !nicknameColumnExists {
		_, err = db.Exec(`ALTER TABLE members ADD COLUMN nickname TEXT`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added nickname column to members table")
	}

	// Create users table
	createUsersTableSQL := `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		member_id INTEGER,
		is_admin BOOLEAN DEFAULT 0,
		FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE SET NULL
	);`

	_, err = db.Exec(createUsersTableSQL)
	if err != nil {
		return err
	}

	// Create train_schedules table
	createTrainSchedulesSQL := `CREATE TABLE IF NOT EXISTS train_schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL UNIQUE,
		conductor_id INTEGER NOT NULL,
		backup_id INTEGER,
		conductor_score INTEGER,
		conductor_showed_up BOOLEAN,
		notes TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (conductor_id) REFERENCES members(id) ON DELETE CASCADE,
		FOREIGN KEY (backup_id) REFERENCES members(id) ON DELETE CASCADE
	);`

	_, err = db.Exec(createTrainSchedulesSQL)
	if err != nil {
		return err
	}

	// Migrate existing train_schedules table to add conductor_score column if missing
	var columnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('train_schedules')
		WHERE name = 'conductor_score'
	`).Scan(&columnExists)
	if err != nil {
		return err
	}

	if !columnExists {
		_, err = db.Exec(`ALTER TABLE train_schedules ADD COLUMN conductor_score INTEGER`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added conductor_score column to train_schedules table")
	}

	// Migrate train_schedules to make backup_id nullable (for existing databases)
	// Check if the table structure needs migration by checking pragma
	migrationNeeded := false
	var backupIdNotnull int
	err = db.QueryRow(`
		SELECT "notnull"
		FROM pragma_table_info('train_schedules')
		WHERE name = 'backup_id'
	`).Scan(&backupIdNotnull)

	if err == nil && backupIdNotnull == 1 {
		migrationNeeded = true
	}

	if migrationNeeded {
		log.Println("Database migration: Making backup_id nullable in train_schedules table")

		// Create new table with correct schema
		_, err = db.Exec(`CREATE TABLE train_schedules_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			conductor_id INTEGER NOT NULL,
			backup_id INTEGER,
			conductor_score INTEGER,
			conductor_showed_up BOOLEAN,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conductor_id) REFERENCES members(id) ON DELETE CASCADE,
			FOREIGN KEY (backup_id) REFERENCES members(id) ON DELETE CASCADE
		)`)
		if err != nil {
			return fmt.Errorf("failed to create new train_schedules table: %v", err)
		}

		// Copy data from old table
		_, err = db.Exec(`INSERT INTO train_schedules_new (id, date, conductor_id, backup_id, conductor_score, conductor_showed_up, notes, created_at)
			SELECT id, date, conductor_id, backup_id, conductor_score, conductor_showed_up, notes, created_at
			FROM train_schedules`)
		if err != nil {
			return fmt.Errorf("failed to copy train_schedules data: %v", err)
		}

		// Drop old table
		_, err = db.Exec(`DROP TABLE train_schedules`)
		if err != nil {
			return fmt.Errorf("failed to drop old train_schedules table: %v", err)
		}

		// Rename new table
		_, err = db.Exec(`ALTER TABLE train_schedules_new RENAME TO train_schedules`)
		if err != nil {
			return fmt.Errorf("failed to rename train_schedules_new table: %v", err)
		}

		log.Println("Database migration: Successfully made backup_id nullable")
	}

	// Migrate existing train_schedules table to add actual_conductor_id column if missing
	var actualConductorColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('train_schedules')
		WHERE name = 'actual_conductor_id'
	`).Scan(&actualConductorColumnExists)
	if err != nil {
		return err
	}

	if !actualConductorColumnExists {
		_, err = db.Exec(`ALTER TABLE train_schedules ADD COLUMN actual_conductor_id INTEGER REFERENCES members(id) ON DELETE SET NULL`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added actual_conductor_id column to train_schedules table")
	}

	// Migrate existing train_schedules table to add vip_id column if missing
	var vipIDColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) > 0 FROM pragma_table_info('train_schedules') WHERE name = 'vip_id'`).Scan(&vipIDColumnExists)
	if err != nil {
		return err
	}
	if !vipIDColumnExists {
		_, err = db.Exec(`ALTER TABLE train_schedules ADD COLUMN vip_id INTEGER REFERENCES members(id) ON DELETE SET NULL`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`ALTER TABLE train_schedules ADD COLUMN vip_name_snapshot TEXT`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added vip_id and vip_name_snapshot columns to train_schedules table")
	}

	// Create award_types table
	createAwardTypesSQL := `CREATE TABLE IF NOT EXISTS award_types (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		active BOOLEAN DEFAULT 1,
		sort_order INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createAwardTypesSQL)
	if err != nil {
		return err
	}

	// Insert default award types if table is empty
	var awardTypeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM award_types").Scan(&awardTypeCount)
	if err != nil {
		return err
	}

	if awardTypeCount == 0 {
		defaultAwards := []string{
			"Alliance Champion",
			"Star of Desert Storm",
			"Soldier Crusher",
			"Divine Healer",
			"Great Destroyer",
			"Grind King",
			"Alliance Exercise MVP",
			"Doom Elite Slayer",
			"Best Manager",
			"Alliance Sponsor",
			"Firefighting Leader",
			"Excavator Radar",
			"Shining Star",
			"MVP",
			"Devil Trainer",
			"Trial Assist King",
			"Good Helper",
		}

		for i, award := range defaultAwards {
			_, err = db.Exec("INSERT INTO award_types (name, active, sort_order) VALUES (?, 1, ?)", award, i)
			if err != nil {
				return err
			}
		}
	}

	// Create awards table
	createAwardsSQL := `CREATE TABLE IF NOT EXISTS awards (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		week_date TEXT NOT NULL,
		award_type TEXT NOT NULL,
		rank INTEGER NOT NULL,
		member_id INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expired BOOLEAN DEFAULT 0,
		FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE,
		UNIQUE(week_date, award_type, rank)
	);`

	_, err = db.Exec(createAwardsSQL)
	if err != nil {
		return err
	}

	// Create power_history table
	createPowerHistorySQL := `CREATE TABLE IF NOT EXISTS power_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		member_id INTEGER NOT NULL,
		power INTEGER NOT NULL,
		recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE
	);`

	_, err = db.Exec(createPowerHistorySQL)
	if err != nil {
		return err
	}

	// Create index for faster power history queries
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_power_history_member ON power_history(member_id, recorded_at DESC)")
	if err != nil {
		return err
	}

	// Create recommendations table
	createRecommendationsSQL := `CREATE TABLE IF NOT EXISTS recommendations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		member_id INTEGER NOT NULL,
		recommended_by_id INTEGER NOT NULL,
		notes TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expired BOOLEAN DEFAULT 0,
		FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE,
		FOREIGN KEY (recommended_by_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	_, err = db.Exec(createRecommendationsSQL)
	if err != nil {
		return err
	}

	// Create conduct reports table (officer notes that expire after 1 week)
	createDynoRecommendationsSQL := `CREATE TABLE IF NOT EXISTS dyno_recommendations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		member_id INTEGER NOT NULL,
		points INTEGER NOT NULL,
		notes TEXT NOT NULL,
		created_by_id INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE,
		FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	_, err = db.Exec(createDynoRecommendationsSQL)
	if err != nil {
		return err
	}

	// Create settings table
	createSettingsSQL := `CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		award_first_points INTEGER NOT NULL DEFAULT 3,
		award_second_points INTEGER NOT NULL DEFAULT 2,
		award_third_points INTEGER NOT NULL DEFAULT 1,
		recommendation_points INTEGER NOT NULL DEFAULT 10,
		recent_conductor_penalty_days INTEGER NOT NULL DEFAULT 30,
		above_average_conductor_penalty INTEGER NOT NULL DEFAULT 10,
		r4r5_rank_boost INTEGER NOT NULL DEFAULT 5,
		first_time_conductor_boost INTEGER NOT NULL DEFAULT 5,
		schedule_message_template TEXT NOT NULL DEFAULT 'Train Schedule - Week {WEEK}\n\n{SCHEDULES}\n\nNext in line:\n{NEXT_3}',
		daily_message_template TEXT,
		power_tracking_enabled BOOLEAN DEFAULT 0,
		server_timezone TEXT NOT NULL DEFAULT 'UTC',
		conductor_time TEXT NOT NULL DEFAULT '15:00',
		backup_time TEXT NOT NULL DEFAULT '16:30',
		display_timezones TEXT NOT NULL DEFAULT '["Europe/London"]',
		alliance_name TEXT NOT NULL DEFAULT 'Last War: Survival',
		alliance_short_name TEXT NOT NULL DEFAULT 'LWS',
		vs_points_daily_target INTEGER NOT NULL DEFAULT 0,
		vs_points_weekly_target INTEGER NOT NULL DEFAULT 0
	);`

	_, err = db.Exec(createSettingsSQL)
	if err != nil {
		return err
	}

	// Create storm assignments table
	createStormAssignmentsSQL := `CREATE TABLE IF NOT EXISTS storm_assignments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_force TEXT NOT NULL CHECK (task_force IN ('A', 'B')),
		building_id TEXT NOT NULL,
		member_id INTEGER NOT NULL,
		position INTEGER NOT NULL CHECK (position BETWEEN 1 AND 4),
		FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE,
		UNIQUE(task_force, building_id, position)
	);`

	_, err = db.Exec(createStormAssignmentsSQL)
	if err != nil {
		return err
	}

	// Create login_sessions table for tracking login history
	createLoginSessionsSQL := `CREATE TABLE IF NOT EXISTS login_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		ip_address TEXT,
		user_agent TEXT,
		country TEXT,
		city TEXT,
		isp TEXT,
		login_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		success BOOLEAN DEFAULT 1,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	_, err = db.Exec(createLoginSessionsSQL)
	if err != nil {
		return err
	}

	// Create index for faster login history queries
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_login_sessions_user ON login_sessions(user_id, login_time DESC)")
	if err != nil {
		return err
	}

	// Create vs_points table for tracking VS points Monday-Saturday
	createVSPointsSQL := `CREATE TABLE IF NOT EXISTS vs_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		member_id INTEGER NOT NULL,
		week_date TEXT NOT NULL,
		monday INTEGER NOT NULL DEFAULT 0,
		tuesday INTEGER NOT NULL DEFAULT 0,
		wednesday INTEGER NOT NULL DEFAULT 0,
		thursday INTEGER NOT NULL DEFAULT 0,
		friday INTEGER NOT NULL DEFAULT 0,
		saturday INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE,
		UNIQUE(member_id, week_date)
	);`

	_, err = db.Exec(createVSPointsSQL)
	if err != nil {
		return err
	}

	// Create index for faster VS points queries
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_vs_points_week ON vs_points(week_date)")
	if err != nil {
		return err
	}

	// Initialize default settings if not exist
	var settingsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&settingsCount)
	if err != nil {
		return err
	}

	if settingsCount == 0 {
		_, err = db.Exec(`INSERT INTO settings (id, award_first_points, award_second_points, award_third_points, 
			recommendation_points, recent_conductor_penalty_days, above_average_conductor_penalty, r4r5_rank_boost, 
			first_time_conductor_boost, schedule_message_template, server_timezone, conductor_time, backup_time, 
			display_timezones, daily_message_template) 
			VALUES (1, 3, 2, 1, 10, 30, 10, 5, 5, 
			'Train Schedule - Week {WEEK}\n\n{SCHEDULES}\n\nNext in line:\n{NEXT_3}', 
			'Etc/GMT+2', '15:00', '16:30', '["Europe/London"]', 
			'Daily train reminder for {DAY}, {DATE}:\n🚂 Conductor: {CONDUCTOR} - Please be online at {CONDUCTOR_TIME}\n🔄 Backup: {BACKUP} - Please be ready at {BACKUP_TIME}\n\nAsk in alliance chat for the train to be assigned. Thanks for keeping the train golden!')`)
		if err != nil {
			return err
		}
		log.Println("Default settings initialized")
	}

	// Migrate settings table to add r4r5_rank_boost column if missing
	var rankBoostColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('settings')
		WHERE name = 'r4r5_rank_boost'
	`).Scan(&rankBoostColumnExists)
	if err != nil {
		return err
	}

	if !rankBoostColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN r4r5_rank_boost INTEGER NOT NULL DEFAULT 5`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added r4r5_rank_boost column to settings table")
	}

	// Migrate settings table to add schedule_message_template column if missing
	var scheduleTemplateColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('settings')
		WHERE name = 'schedule_message_template'
	`).Scan(&scheduleTemplateColumnExists)
	if err != nil {
		return err
	}

	if !scheduleTemplateColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN schedule_message_template TEXT NOT NULL DEFAULT 'Train Schedule - Week {WEEK}

{SCHEDULES}

Next in line:
{NEXT_3}'`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added schedule_message_template column to settings table")
	}

	// Migrate settings table to add first_time_conductor_boost column if missing
	var firstTimeBoostColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('settings')
		WHERE name = 'first_time_conductor_boost'
	`).Scan(&firstTimeBoostColumnExists)
	if err != nil {
		return err
	}

	if !firstTimeBoostColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN first_time_conductor_boost INTEGER NOT NULL DEFAULT 5`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added first_time_conductor_boost column to settings table")
	}

	// Migrate settings table to add daily_message_template column if missing
	var dailyTemplateColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('settings')
		WHERE name = 'daily_message_template'
	`).Scan(&dailyTemplateColumnExists)
	if err != nil {
		return err
	}

	if !dailyTemplateColumnExists {
		defaultDailyTemplate := `Daily train reminder for {DAY}, {DATE}:
🚂 Conductor: {CONDUCTOR_NAME} - Please be online at {CONDUCTOR_TIME}
🔄 Backup: {BACKUP_NAME} - Please be ready at {BACKUP_TIME}

Ask in alliance chat for the train to be assigned. Thanks for keeping the train golden!`
		// Add column without default first
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN daily_message_template TEXT`)
		if err != nil {
			return err
		}
		// Then update existing row with the default value
		_, err = db.Exec(`UPDATE settings SET daily_message_template = ? WHERE id = 1`, defaultDailyTemplate)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added daily_message_template column to settings table")
	}

	// Migrate settings table to add power_tracking_enabled column if missing
	var powerTrackingColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('settings') 
		WHERE name='power_tracking_enabled'
	`).Scan(&powerTrackingColumnExists)
	if err != nil || !powerTrackingColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN power_tracking_enabled BOOLEAN DEFAULT 0`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added power_tracking_enabled column to settings table")
	}

	// Migrate settings table to add server_timezone column if missing
	var timezoneColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('settings') 
		WHERE name='server_timezone'
	`).Scan(&timezoneColumnExists)
	if err != nil || !timezoneColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN server_timezone TEXT NOT NULL DEFAULT 'UTC'`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added server_timezone column to settings table")
	}

	// Migrate settings table to add conductor_time column if missing
	var conductorTimeColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('settings') 
		WHERE name='conductor_time'
	`).Scan(&conductorTimeColumnExists)
	if err != nil || !conductorTimeColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN conductor_time TEXT NOT NULL DEFAULT '15:00'`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added conductor_time column to settings table")
	}

	// Migrate settings table to add backup_time column if missing
	var backupTimeColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('settings') 
		WHERE name='backup_time'
	`).Scan(&backupTimeColumnExists)
	if err != nil || !backupTimeColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN backup_time TEXT NOT NULL DEFAULT '16:30'`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added backup_time column to settings table")
	}

	// Migrate settings table to add display_timezones column if missing
	var displayTimezonesColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('settings') 
		WHERE name='display_timezones'
	`).Scan(&displayTimezonesColumnExists)
	if err != nil || !displayTimezonesColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN display_timezones TEXT NOT NULL DEFAULT '["Europe/London"]'`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added display_timezones column to settings table")
	}

	// Migrate settings table to add alliance_name column if missing
	var allianceNameColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('settings') 
		WHERE name='alliance_name'
	`).Scan(&allianceNameColumnExists)
	if err != nil || !allianceNameColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN alliance_name TEXT NOT NULL DEFAULT 'Last War: Survival'`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added alliance_name column to settings table")
	}

	// Migrate settings table to add vs_points targets columns if missing
	var vsDailyTargetColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('settings') WHERE name='vs_points_daily_target'`).Scan(&vsDailyTargetColumnExists)
	if err != nil || !vsDailyTargetColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN vs_points_daily_target INTEGER NOT NULL DEFAULT 0`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN vs_points_weekly_target INTEGER NOT NULL DEFAULT 0`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added vs_points_daily_target and vs_points_weekly_target columns to settings table")
	}

	// Migrate settings table to add backup_rotation_order column if missing
	var backupRotationColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('settings') WHERE name='backup_rotation_order'`).Scan(&backupRotationColumnExists)
	if err != nil || !backupRotationColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN backup_rotation_order TEXT NOT NULL DEFAULT '[]'`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added backup_rotation_order column to settings table")
	}

	// Migrate settings table to add train_week_mode column if missing
	var trainWeekModeColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('settings') WHERE name='train_week_mode'`).Scan(&trainWeekModeColumnExists)
	if err != nil || !trainWeekModeColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN train_week_mode TEXT NOT NULL DEFAULT 'win'`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added train_week_mode column to settings table")
	}

	// Migrate settings table to add alliance_short_name column if missing
	var allianceShortNameColumnExists bool
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('settings') 
		WHERE name='alliance_short_name'
	`).Scan(&allianceShortNameColumnExists)
	if err != nil || !allianceShortNameColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN alliance_short_name TEXT NOT NULL DEFAULT 'LWS'`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added alliance_short_name column to settings table")
	}

	// Create default admin user if no users exist
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return err
	}

	if userCount == 0 {
		// Default credentials: admin/admin123
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = db.Exec("INSERT INTO users (username, password, is_admin, must_change_password) VALUES (?, ?, ?, ?)", "admin", string(hashedPassword), true, true)
		if err != nil {
			return err
		}
		log.Println("Default admin user created - Username: admin, Password: admin123 (password change required)")
	}

	// Migrate members table to add soft-delete columns
	var deletedAtColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) > 0 FROM pragma_table_info('members') WHERE name = 'deleted_at'`).Scan(&deletedAtColumnExists)
	if err != nil {
		return err
	}
	if !deletedAtColumnExists {
		_, err = db.Exec(`ALTER TABLE members ADD COLUMN deleted_at TIMESTAMP`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`ALTER TABLE members ADD COLUMN deleted_by_id INTEGER REFERENCES users(id) ON DELETE SET NULL`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`ALTER TABLE members ADD COLUMN deletion_reason TEXT`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added soft-delete columns to members table")
	}

	// Partial unique index: only active (non-deleted) members must have unique names
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_members_name_active ON members(name) WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}

	// Migrate users table to add active column (for suspending accounts when member is deleted)
	var userActiveColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) > 0 FROM pragma_table_info('users') WHERE name = 'active'`).Scan(&userActiveColumnExists)
	if err != nil {
		return err
	}
	if !userActiveColumnExists {
		_, err = db.Exec(`ALTER TABLE users ADD COLUMN active BOOLEAN NOT NULL DEFAULT 1`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added active column to users table")
	}

	// Migrate users table to add must_change_password column
	var mustChangePwdExists bool
	err = db.QueryRow(`SELECT COUNT(*) > 0 FROM pragma_table_info('users') WHERE name = 'must_change_password'`).Scan(&mustChangePwdExists)
	if err != nil {
		return err
	}
	if !mustChangePwdExists {
		_, err = db.Exec(`ALTER TABLE users ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT 0`)
		if err != nil {
			return err
		}
		// Flag existing admin accounts that still have the default password
		rows, qErr := db.Query(`SELECT id, password FROM users WHERE username = 'admin' AND is_admin = 1`)
		if qErr == nil {
			defer rows.Close()
			for rows.Next() {
				var uid int
				var hash string
				if rows.Scan(&uid, &hash) == nil {
					if bcrypt.CompareHashAndPassword([]byte(hash), []byte("admin123")) == nil {
						db.Exec(`UPDATE users SET must_change_password = 1 WHERE id = ?`, uid)
						log.Println("Flagged default admin account for mandatory password change")
					}
				}
			}
		}
		log.Println("Database migration: Added must_change_password column to users table")
	}

	// Migrate train_schedules to add name snapshot columns (preserved even after member is deleted)
	var conductorSnapshotExists bool
	err = db.QueryRow(`SELECT COUNT(*) > 0 FROM pragma_table_info('train_schedules') WHERE name = 'conductor_name_snapshot'`).Scan(&conductorSnapshotExists)
	if err != nil {
		return err
	}
	if !conductorSnapshotExists {
		_, err = db.Exec(`ALTER TABLE train_schedules ADD COLUMN conductor_name_snapshot TEXT`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`ALTER TABLE train_schedules ADD COLUMN backup_name_snapshot TEXT`)
		if err != nil {
			return err
		}
		// Backfill snapshots from current member names
		_, err = db.Exec(`UPDATE train_schedules SET
			conductor_name_snapshot = (SELECT name FROM members WHERE id = conductor_id),
			backup_name_snapshot = (SELECT name FROM members WHERE id = backup_id)`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added name snapshot columns to train_schedules table")
	}

	// Migrate awards to add member name snapshot column
	var awardSnapshotExists bool
	err = db.QueryRow(`SELECT COUNT(*) > 0 FROM pragma_table_info('awards') WHERE name = 'member_name_snapshot'`).Scan(&awardSnapshotExists)
	if err != nil {
		return err
	}
	if !awardSnapshotExists {
		_, err = db.Exec(`ALTER TABLE awards ADD COLUMN member_name_snapshot TEXT`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE awards SET member_name_snapshot = (SELECT name FROM members WHERE id = member_id)`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added member_name_snapshot column to awards table")
	}

	// Migrate recommendations to add member name snapshot column
	var recSnapshotExists bool
	err = db.QueryRow(`SELECT COUNT(*) > 0 FROM pragma_table_info('recommendations') WHERE name = 'member_name_snapshot'`).Scan(&recSnapshotExists)
	if err != nil {
		return err
	}
	if !recSnapshotExists {
		_, err = db.Exec(`ALTER TABLE recommendations ADD COLUMN member_name_snapshot TEXT`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE recommendations SET member_name_snapshot = (SELECT name FROM members WHERE id = member_id)`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added member_name_snapshot column to recommendations table")
	}

	// Migrate dyno_recommendations (conduct reports) to add member name snapshot column
	var dynoSnapshotExists bool
	err = db.QueryRow(`SELECT COUNT(*) > 0 FROM pragma_table_info('dyno_recommendations') WHERE name = 'member_name_snapshot'`).Scan(&dynoSnapshotExists)
	if err != nil {
		return err
	}
	if !dynoSnapshotExists {
		_, err = db.Exec(`ALTER TABLE dyno_recommendations ADD COLUMN member_name_snapshot TEXT`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE dyno_recommendations SET member_name_snapshot = (SELECT name FROM members WHERE id = member_id)`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added member_name_snapshot column to dyno_recommendations table")
	}

	// Create applicants table (recruitment tracker)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS applicants (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		power INTEGER,
		rank TEXT,
		vouched_by TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		decision_by_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		decided_at TIMESTAMP,
		trial_end_date DATE,
		notes TEXT,
		rejection_reason TEXT,
		member_id INTEGER REFERENCES members(id) ON DELETE SET NULL
	)`)
	if err != nil {
		return err
	}

	// Migrate settings table to add min_power column if missing
	var minPowerColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('settings') WHERE name='min_power'`).Scan(&minPowerColumnExists)
	if err != nil || !minPowerColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN min_power INTEGER NOT NULL DEFAULT 0`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added min_power column to settings table")
	}

	// Migrate settings table to add min_hq_level column if missing
	var minHQColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('settings') WHERE name='min_hq_level'`).Scan(&minHQColumnExists)
	if err != nil || !minHQColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN min_hq_level INTEGER NOT NULL DEFAULT 0`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added min_hq_level column to settings table")
	}

	// Migrate settings table to add vip_seat_enabled column if missing
	var vipSeatEnabledColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('settings') WHERE name='vip_seat_enabled'`).Scan(&vipSeatEnabledColumnExists)
	if err != nil || !vipSeatEnabledColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN vip_seat_enabled INTEGER NOT NULL DEFAULT 1`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added vip_seat_enabled column to settings table")
	}

	// Migrate settings table to add marshal_guard_enabled column if missing
	var mgEnabledColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('settings') WHERE name='marshal_guard_enabled'`).Scan(&mgEnabledColumnExists)
	if err != nil || !mgEnabledColumnExists {
		_, err = db.Exec(`ALTER TABLE settings ADD COLUMN marshal_guard_enabled INTEGER NOT NULL DEFAULT 1`)
		if err != nil {
			return err
		}
		log.Println("Database migration: Added marshal_guard_enabled column to settings table")
	}

	// Create marshal_guard_events table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS marshal_guard_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_date TEXT NOT NULL,
		total_alliance_damage INTEGER NOT NULL DEFAULT 0,
		notes TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_by_id INTEGER REFERENCES users(id)
	)`)
	if err != nil {
		return err
	}

	// Create marshal_guard_participants table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS marshal_guard_participants (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id INTEGER NOT NULL REFERENCES marshal_guard_events(id) ON DELETE CASCADE,
		member_id INTEGER REFERENCES members(id),
		name_snapshot TEXT NOT NULL,
		alliance_tag TEXT,
		rank_in_event INTEGER NOT NULL,
		damage INTEGER NOT NULL DEFAULT 0,
		attack_count INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(event_id, rank_in_event)
	)`)
	if err != nil {
		return err
	}

	return nil
}

// Authentication middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Block all API access except password change and auth check when password change is required
		if mustChange, ok := session.Values["must_change_password"].(bool); ok && mustChange {
			path := r.URL.Path
			if path != "/api/change-password" && path != "/api/check-auth" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":                "Password change required",
					"must_change_password": true,
				})
				return
			}
		}
		next(w, r)
	}
}

// Permission middleware - only R4/R5 or admin can manage ranks
func rankManagementMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")

		// Check if admin
		if isAdmin, ok := session.Values["is_admin"].(bool); ok && isAdmin {
			next(w, r)
			return
		}

		// Check if member has R4 or R5 rank
		if memberID, ok := session.Values["member_id"].(int); ok {
			var rank string
			err := db.QueryRow("SELECT rank FROM members WHERE id = ? AND deleted_at IS NULL", memberID).Scan(&rank)
			if err == nil && (rank == "R4" || rank == "R5") {
				next(w, r)
				return
			}
		}

		http.Error(w, "Forbidden: Only R4/R5 members can manage ranks", http.StatusForbidden)
	}
}

// r3PlusMiddleware allows R3, R4, R5, and admin (used for MG upload/confirm).
func r3PlusMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		if isAdmin, ok := session.Values["is_admin"].(bool); ok && isAdmin {
			next(w, r)
			return
		}
		if memberID, ok := session.Values["member_id"].(int); ok {
			var rank string
			if err := db.QueryRow("SELECT rank FROM members WHERE id = ? AND deleted_at IS NULL", memberID).Scan(&rank); err == nil {
				if rank == "R3" || rank == "R4" || rank == "R5" {
					next(w, r)
					return
				}
			}
		}
		http.Error(w, "Forbidden: R3 or higher required", http.StatusForbidden)
	}
}

// Permission middleware - only R5 or admin
func adminR5Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")

		// Check if admin
		if isAdmin, ok := session.Values["is_admin"].(bool); ok && isAdmin {
			next(w, r)
			return
		}

		// Check if member has R5 rank
		if memberID, ok := session.Values["member_id"].(int); ok {
			var rank string
			err := db.QueryRow("SELECT rank FROM members WHERE id = ? AND deleted_at IS NULL", memberID).Scan(&rank)
			if err == nil && rank == "R5" {
				next(w, r)
				return
			}
		}

		http.Error(w, "Forbidden: Only R5 members and admins can perform this action", http.StatusForbidden)
	}
}

// Admin-only middleware
func adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")

		if isAdmin, ok := session.Values["is_admin"].(bool); !ok || !isAdmin {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// Login handler
func login(w http.ResponseWriter, r *http.Request) {
	// Rate limiting check
	clientIP := getClientIP(r)
	if rateLimiter.isRateLimited(clientIP) {
		http.Error(w, "Too many failed login attempts. Please try again later.", http.StatusTooManyRequests)
		return
	}

	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var user User
	var memberID sql.NullInt64
	var isAdmin sql.NullBool
	var mustChangePwd bool
	err := db.QueryRow("SELECT id, username, password, member_id, is_admin, COALESCE(must_change_password, 0) FROM users WHERE username = ?", creds.Username).Scan(&user.ID, &user.Username, &user.Password, &memberID, &isAdmin, &mustChangePwd)
	if err != nil {
		// Track failed login attempt
		rateLimiter.recordFailure(clientIP)
		trackLogin(0, creds.Username, r, false)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if memberID.Valid {
		mid := int(memberID.Int64)
		user.MemberID = &mid
	}
	user.IsAdmin = isAdmin.Valid && isAdmin.Bool

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password))
	if err != nil {
		// Track failed login attempt
		rateLimiter.recordFailure(clientIP)
		trackLogin(user.ID, user.Username, r, false)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check if account is active (not suspended due to member deletion)
	var isActive bool
	if scanErr := db.QueryRow("SELECT COALESCE(active, 1) FROM users WHERE id = ?", user.ID).Scan(&isActive); scanErr == nil && !isActive {
		trackLogin(user.ID, user.Username, r, false)
		http.Error(w, "Account suspended", http.StatusForbidden)
		return
	}

	// Track successful login
	trackLogin(user.ID, user.Username, r, true)

	// Create session
	session, _ := store.Get(r, "session")
	session.Values["authenticated"] = true
	session.Values["username"] = user.Username
	session.Values["user_id"] = user.ID
	if user.MemberID != nil {
		session.Values["member_id"] = *user.MemberID
	}
	session.Values["is_admin"] = user.IsAdmin
	session.Values["must_change_password"] = mustChangePwd
	session.Save(r, w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Login successful", "username": user.Username, "must_change_password": mustChangePwd})
}

// Logout handler
func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1
	session.Save(r, w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Logout successful"})
}

// Change password handler
func changePassword(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(input.NewPassword) < 6 {
		http.Error(w, "New password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// Get current password hash
	var currentHash string
	err := db.QueryRow("SELECT password FROM users WHERE id = ?", userID).Scan(&currentHash)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(input.CurrentPassword))
	if err != nil {
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Update password and clear must_change_password flag
	_, err = db.Exec("UPDATE users SET password = ?, must_change_password = 0 WHERE id = ?", string(newHash), userID)
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	// Clear the flag in the session
	session.Values["must_change_password"] = false
	session.Save(r, w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Password changed successfully"})
}

// Generate random alphanumeric password
func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, length)
	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[num.Int64()]
	}
	return string(password), nil
}

// Get client IP with X-Forwarded-For and X-Real-IP support
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header (nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return ip
}

// Get IP geolocation information using ip-api.com (free, no key required)
func getIPGeolocation(ip string) (*IPGeolocation, error) {
	// Skip localhost/private IPs
	if ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
		return &IPGeolocation{
			Status:  "success",
			Country: "Local Network",
			City:    "Localhost",
			ISP:     "Private Network",
			Query:   ip,
		}, nil
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,region,regionName,city,zip,lat,lon,timezone,isp,org,as,query", ip)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var geo IPGeolocation
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return nil, err
	}

	if geo.Status != "success" {
		return nil, fmt.Errorf("geolocation lookup failed")
	}

	return &geo, nil
}

// Track login attempt in database
func trackLogin(userID int, username string, r *http.Request, success bool) {
	ip := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	var country, city, isp *string

	// Get geolocation data (non-blocking, log errors but don't fail)
	if geo, err := getIPGeolocation(ip); err == nil {
		country = &geo.Country
		city = &geo.City
		isp = &geo.ISP
	}

	_, err := db.Exec(`INSERT INTO login_sessions (user_id, username, ip_address, user_agent, country, city, isp, success) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, username, ip, userAgent, country, city, isp, success)

	if err != nil {
		log.Printf("Failed to track login: %v", err)
	}
}

// Create user for member
func createUserForMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memberID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	// Check if member exists and is active
	var memberName string
	err = db.QueryRow("SELECT name FROM members WHERE id = ? AND deleted_at IS NULL", memberID).Scan(&memberName)
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	// Check if user already exists for this member
	var existingUserID int
	err = db.QueryRow("SELECT id FROM users WHERE member_id = ?", memberID).Scan(&existingUserID)
	if err == nil {
		http.Error(w, "User already exists for this member", http.StatusConflict)
		return
	}

	// Generate random password
	randomPassword, err := generateRandomPassword(10)
	if err != nil {
		http.Error(w, "Failed to generate password", http.StatusInternalServerError)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Create username from member name (lowercase, no spaces)
	username := strings.ToLower(strings.ReplaceAll(memberName, " ", ""))

	// Check if username already exists, if so, append member ID
	var existingUsername string
	err = db.QueryRow("SELECT username FROM users WHERE username = ?", username).Scan(&existingUsername)
	if err == nil {
		username = username + strconv.Itoa(memberID)
	}

	// Insert user
	_, err = db.Exec("INSERT INTO users (username, password, member_id, is_admin) VALUES (?, ?, ?, ?)",
		username, string(hashedPassword), memberID, false)
	if err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "User created successfully",
		"username": username,
		"password": randomPassword,
	})
}

// Check auth status
func checkAuth(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	if auth, ok := session.Values["authenticated"].(bool); ok && auth {
		username := session.Values["username"].(string)
		isAdmin := false
		if adminVal, ok := session.Values["is_admin"].(bool); ok {
			isAdmin = adminVal
		}

		var rank string
		var canManageRanks bool

		if isAdmin {
			rank = "Admin"
			canManageRanks = true
		} else if memberID, ok := session.Values["member_id"].(int); ok {
			// Get member's rank
			err := db.QueryRow("SELECT rank FROM members WHERE id = ?", memberID).Scan(&rank)
			if err == nil {
				canManageRanks = (rank == "R4" || rank == "R5")
			}
		}

		// Check if user is R5 or admin (for more sensitive operations)
		isR5OrAdmin := isAdmin
		if !isR5OrAdmin && rank == "R5" {
			isR5OrAdmin = true
		}

		mustChangePwd := false
		if v, ok := session.Values["must_change_password"].(bool); ok {
			mustChangePwd = v
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"authenticated":        true,
			"username":             username,
			"rank":                 rank,
			"is_admin":             isAdmin,
			"can_manage_ranks":     canManageRanks,
			"is_r5_or_admin":       isR5OrAdmin,
			"must_change_password": mustChangePwd,
		}
		if memberID, ok := session.Values["member_id"].(int); ok {
			response["member_id"] = memberID
		}
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
	}
}

// Admin: Get all users with login information
func getAdminUsers(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT u.id, u.username, u.member_id, u.is_admin, 
			   m.name as member_name,
			   (SELECT login_time FROM login_sessions WHERE user_id = u.id AND success = 1 ORDER BY login_time DESC LIMIT 1) as last_login,
			   (SELECT COUNT(*) FROM login_sessions WHERE user_id = u.id AND success = 1) as login_count
		FROM users u
		LEFT JOIN members m ON u.member_id = m.id
		ORDER BY u.is_admin DESC, u.username ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []AdminUserResponse{}
	for rows.Next() {
		var user AdminUserResponse
		var memberID sql.NullInt64
		var memberName sql.NullString
		var lastLogin sql.NullString

		err := rows.Scan(&user.ID, &user.Username, &memberID, &user.IsAdmin,
			&memberName, &lastLogin, &user.LoginCount)
		if err != nil {
			continue
		}

		if memberID.Valid {
			mid := int(memberID.Int64)
			user.MemberID = &mid
		}
		if memberName.Valid {
			user.MemberName = &memberName.String
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.String
		}

		// Get recent logins (last 5)
		loginRows, err := db.Query(`
			SELECT id, user_id, username, ip_address, user_agent, country, city, isp, login_time, success
			FROM login_sessions
			WHERE user_id = ? AND success = 1
			ORDER BY login_time DESC
			LIMIT 5
		`, user.ID)

		if err == nil {
			recentLogins := []LoginSession{}
			for loginRows.Next() {
				var login LoginSession
				var ipAddr, userAgent, country, city, isp sql.NullString

				loginRows.Scan(&login.ID, &login.UserID, &login.Username,
					&ipAddr, &userAgent, &country, &city, &isp,
					&login.LoginTime, &login.Success)

				if ipAddr.Valid {
					login.IPAddress = &ipAddr.String
				}
				if userAgent.Valid {
					login.UserAgent = &userAgent.String
				}
				if country.Valid {
					login.Country = &country.String
				}
				if city.Valid {
					login.City = &city.String
				}
				if isp.Valid {
					login.ISP = &isp.String
				}

				recentLogins = append(recentLogins, login)
			}
			loginRows.Close()
			user.RecentLogins = recentLogins
		}

		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Admin: Create new user
func createAdminUser(w http.ResponseWriter, r *http.Request) {
	var req AdminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	var existingID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", req.Username).Scan(&existingID)
	if err == nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Insert user
	result, err := db.Exec("INSERT INTO users (username, password, member_id, is_admin) VALUES (?, ?, ?, ?)",
		req.Username, string(hashedPassword), req.MemberID, req.IsAdmin)
	if err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User created successfully",
		"id":      id,
	})
}

// Admin: Update user
func updateAdminUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req AdminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if user exists
	var existingUsername string
	err = db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&existingUsername)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if new username already exists (if username is being changed)
	if req.Username != "" && req.Username != existingUsername {
		var otherID int
		err = db.QueryRow("SELECT id FROM users WHERE username = ? AND id != ?", req.Username, userID).Scan(&otherID)
		if err == nil {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}
	}

	// Build update query
	if req.Username != "" {
		_, err = db.Exec("UPDATE users SET username = ?, member_id = ?, is_admin = ? WHERE id = ?",
			req.Username, req.MemberID, req.IsAdmin, userID)
	} else {
		_, err = db.Exec("UPDATE users SET member_id = ?, is_admin = ? WHERE id = ?",
			req.MemberID, req.IsAdmin, userID)
	}

	if err != nil {
		http.Error(w, "Failed to update user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User updated successfully"})
}

// Admin: Delete user
func deleteAdminUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if user exists
	var username string
	err = db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Prevent deleting the last admin
	var adminCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&adminCount)
	if err == nil && adminCount <= 1 {
		var isAdmin bool
		db.QueryRow("SELECT is_admin FROM users WHERE id = ?", userID).Scan(&isAdmin)
		if isAdmin {
			http.Error(w, "Cannot delete the last admin user", http.StatusForbidden)
			return
		}
	}

	// Delete user
	_, err = db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		http.Error(w, "Failed to delete user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User deleted successfully"})
}

// Admin: Reset user password
func resetUserPassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if user exists
	var username string
	err = db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Generate random password
	randomPassword, err := generateRandomPassword(10)
	if err != nil {
		http.Error(w, "Failed to generate password", http.StatusInternalServerError)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Update password
	_, err = db.Exec("UPDATE users SET password = ? WHERE id = ?", string(hashedPassword), userID)
	if err != nil {
		http.Error(w, "Failed to reset password: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Password reset successfully",
		"username": username,
		"password": randomPassword,
	})
}

// Admin: Get login history
func getLoginHistory(w http.ResponseWriter, r *http.Request) {
	// Get optional filters from query params
	userIDParam := r.URL.Query().Get("user_id")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	// Validate limit is a positive integer to prevent injection
	limitNum, err := strconv.Atoi(limit)
	if err != nil || limitNum <= 0 || limitNum > 10000 {
		limitNum = 100
	}

	query := `
		SELECT ls.id, ls.user_id, ls.username, ls.ip_address, ls.user_agent, 
		       ls.country, ls.city, ls.isp, ls.login_time, ls.success
		FROM login_sessions ls
	`

	var args []interface{}
	if userIDParam != "" {
		query += " WHERE ls.user_id = ?"
		args = append(args, userIDParam)
	}

	query += " ORDER BY ls.login_time DESC LIMIT ?"
	args = append(args, limitNum)

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch login history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	history := []LoginSession{}
	for rows.Next() {
		var login LoginSession
		var ipAddr, userAgent, country, city, isp sql.NullString

		err := rows.Scan(&login.ID, &login.UserID, &login.Username,
			&ipAddr, &userAgent, &country, &city, &isp,
			&login.LoginTime, &login.Success)
		if err != nil {
			continue
		}

		if ipAddr.Valid {
			login.IPAddress = &ipAddr.String
		}
		if userAgent.Valid {
			login.UserAgent = &userAgent.String
		}
		if country.Valid {
			login.Country = &country.String
		}
		if city.Valid {
			login.City = &city.String
		}
		if isp.Valid {
			login.ISP = &isp.String
		}

		history = append(history, login)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// Get all members
func getMembers(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT m.id, m.name, m.nickname, m.rank, COALESCE(m.eligible, 1),
		       (SELECT ph.power 
		        FROM power_history ph 
		        WHERE ph.member_id = m.id 
		        ORDER BY ph.recorded_at DESC 
		        LIMIT 1) as latest_power
		FROM members m
		WHERE m.deleted_at IS NULL
		ORDER BY m.name
	`
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	members := []Member{}
	for rows.Next() {
		var m Member
		var nickname sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &nickname, &m.Rank, &m.Eligible, &m.Power); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if nickname.Valid && nickname.String != "" {
			m.Nickname = &nickname.String
		}
		members = append(members, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// Get member statistics for train scheduling
func getMemberStats(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT 
			m.id, 
			m.name, 
			m.rank,
			COUNT(DISTINCT CASE WHEN ts.conductor_id = m.id THEN ts.date END) as conductor_count,
			MAX(CASE WHEN ts.conductor_id = m.id THEN ts.date END) as last_conductor_date,
			COUNT(DISTINCT CASE WHEN ts.backup_id = m.id THEN ts.date END) as backup_count,
			COUNT(DISTINCT CASE WHEN ts.backup_id = m.id AND ts.conductor_showed_up = 0 AND ts.actual_conductor_id IS NULL THEN ts.date END) as backup_used_count,
			COUNT(DISTINCT CASE WHEN ts.actual_conductor_id = m.id THEN ts.date END) as actual_conductor_count,
			COUNT(DISTINCT CASE WHEN ts.conductor_id = m.id AND ts.conductor_showed_up = 0 THEN ts.date END) as conductor_no_show_count
		FROM members m
		LEFT JOIN train_schedules ts ON ts.conductor_id = m.id OR ts.backup_id = m.id OR ts.actual_conductor_id = m.id
		WHERE m.deleted_at IS NULL
		GROUP BY m.id, m.name, m.rank
		ORDER BY m.name
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	stats := []MemberStats{}
	for rows.Next() {
		var s MemberStats
		var lastDate sql.NullString
		if err := rows.Scan(&s.ID, &s.Name, &s.Rank, &s.ConductorCount, &lastDate, &s.BackupCount, &s.BackupUsedCount, &s.ActualConductorCount, &s.ConductorNoShowCount); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if lastDate.Valid {
			s.LastConductorDate = &lastDate.String
		}
		stats = append(stats, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Create a new member
func createMember(w http.ResponseWriter, r *http.Request) {
	var m Member
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Default to eligible if not specified
	if !m.Eligible {
		m.Eligible = true
	}

	var nicknameVal interface{}
	if m.Nickname != nil && *m.Nickname != "" {
		nicknameVal = *m.Nickname
	}

	result, err := db.Exec("INSERT INTO members (name, nickname, rank, eligible) VALUES (?, ?, ?, ?)", m.Name, nicknameVal, m.Rank, m.Eligible)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	m.ID = int(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m)
}

// Update a member
func updateMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	var m Member
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var nicknameVal interface{}
	if m.Nickname != nil && *m.Nickname != "" {
		nicknameVal = *m.Nickname
	}

	_, err = db.Exec("UPDATE members SET name = ?, nickname = ?, rank = ?, eligible = ? WHERE id = ?", m.Name, nicknameVal, m.Rank, m.Eligible, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	m.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

// Delete a member
func deleteMember(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&input) // reason is optional

	deleterUserID, _ := session.Values["user_id"].(int)
	today := formatDateString(getServerTime())

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Soft-delete the member
	var reasonVal interface{}
	if input.Reason != "" {
		reasonVal = input.Reason
	}
	result, err := tx.Exec(
		"UPDATE members SET deleted_at = CURRENT_TIMESTAMP, deleted_by_id = ?, deletion_reason = ? WHERE id = ? AND deleted_at IS NULL",
		deleterUserID, reasonVal, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		http.Error(w, "Member not found or already deleted", http.StatusNotFound)
		return
	}

	// Expire all open recommendations for this member
	tx.Exec("UPDATE recommendations SET expired = 1 WHERE member_id = ?", id)

	// Remove from future conductor slots (can't have a schedule without a conductor)
	tx.Exec("DELETE FROM train_schedules WHERE conductor_id = ? AND date >= ?", id, today)

	// Clear from future backup slots (schedule can survive without a backup)
	tx.Exec("UPDATE train_schedules SET backup_id = NULL, backup_name_snapshot = NULL WHERE backup_id = ? AND date >= ?", id, today)

	// Suspend the linked user account (if any)
	tx.Exec("UPDATE users SET active = 0 WHERE member_id = ?", id)

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to delete member", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Get archived (soft-deleted) members
func getArchivedMembers(w http.ResponseWriter, r *http.Request) {
	type ArchivedMember struct {
		ID             int     `json:"id"`
		Name           string  `json:"name"`
		Rank           string  `json:"rank"`
		DeletedAt      string  `json:"deleted_at"`
		DeletionReason *string `json:"deletion_reason,omitempty"`
		DeletedBy      *string `json:"deleted_by,omitempty"`
		TrainCount     int     `json:"train_count"`
		AwardCount     int     `json:"award_count"`
	}

	rows, err := db.Query(`
		SELECT m.id, m.name, m.rank, m.deleted_at, m.deletion_reason,
			u.username as deleted_by,
			(SELECT COUNT(*) FROM train_schedules ts WHERE ts.conductor_id = m.id OR ts.actual_conductor_id = m.id) as train_count,
			(SELECT COUNT(*) FROM awards a WHERE a.member_id = m.id) as award_count
		FROM members m
		LEFT JOIN users u ON m.deleted_by_id = u.id
		WHERE m.deleted_at IS NOT NULL
		ORDER BY m.deleted_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var members []ArchivedMember
	for rows.Next() {
		var am ArchivedMember
		if err := rows.Scan(&am.ID, &am.Name, &am.Rank, &am.DeletedAt, &am.DeletionReason, &am.DeletedBy, &am.TrainCount, &am.AwardCount); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		members = append(members, am)
	}
	if members == nil {
		members = []ArchivedMember{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// Restore a soft-deleted member
func restoreMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	// Check the member exists and is deleted
	var memberName string
	err = db.QueryRow("SELECT name FROM members WHERE id = ? AND deleted_at IS NOT NULL", id).Scan(&memberName)
	if err == sql.ErrNoRows {
		http.Error(w, "Archived member not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check for name conflict with an active member
	var conflictID int
	err = db.QueryRow("SELECT id FROM members WHERE name = ? AND deleted_at IS NULL", memberName).Scan(&conflictID)
	if err == nil {
		http.Error(w, "An active member with this name already exists", http.StatusConflict)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Restore the member
	_, err = tx.Exec("UPDATE members SET deleted_at = NULL, deleted_by_id = NULL, deletion_reason = NULL WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Reactivate linked user account
	tx.Exec("UPDATE users SET active = 1 WHERE member_id = ?", id)

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to restore member", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   id,
		"name": memberName,
	})
}

// Permanently delete a soft-deleted member and all their data
func permanentlyDeleteMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	// Only allow deleting members that are already soft-deleted
	var memberName string
	err = db.QueryRow("SELECT name FROM members WHERE id = ? AND deleted_at IS NOT NULL", id).Scan(&memberName)
	if err == sql.ErrNoRows {
		http.Error(w, "Archived member not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete all associated data before removing the member row
	tx.Exec("DELETE FROM train_schedules WHERE conductor_id = ? OR backup_id = ? OR actual_conductor_id = ?", id, id, id)
	tx.Exec("DELETE FROM power_records WHERE member_id = ?", id)
	tx.Exec("DELETE FROM awards WHERE member_id = ?", id)
	tx.Exec("DELETE FROM recommendations WHERE member_id = ?", id)
	tx.Exec("DELETE FROM dyno_recommendations WHERE member_id = ?", id)
	tx.Exec("DELETE FROM vs_points WHERE member_id = ?", id)
	tx.Exec("DELETE FROM users WHERE member_id = ?", id)
	_, err = tx.Exec("DELETE FROM members WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Failed to delete member: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit deletion", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Get train schedules (optionally filtered by date range)
func getTrainSchedules(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")

	query := `
		SELECT 
			ts.id, ts.date, ts.conductor_id,
			COALESCE(m1.name, ts.conductor_name_snapshot, '[Removed Member]') as conductor_name,
			ts.conductor_score,
			COALESCE(ts.backup_id, 0) as backup_id,
			COALESCE(m2.name, ts.backup_name_snapshot, '') as backup_name,
			COALESCE(m2.rank, '') as backup_rank,
			ts.conductor_showed_up, ts.actual_conductor_id,
			COALESCE(m3.name, '[Removed Member]') as actual_conductor_name,
			ts.notes, ts.created_at,
			ts.vip_id,
			COALESCE(m4.name, ts.vip_name_snapshot, '') as vip_name
		FROM train_schedules ts
		LEFT JOIN members m1 ON ts.conductor_id = m1.id AND m1.deleted_at IS NULL
		LEFT JOIN members m2 ON ts.backup_id = m2.id AND m2.deleted_at IS NULL
		LEFT JOIN members m3 ON ts.actual_conductor_id = m3.id
		LEFT JOIN members m4 ON ts.vip_id = m4.id AND m4.deleted_at IS NULL
	`

	var rows *sql.Rows
	var err error

	if startDate != "" && endDate != "" {
		query += " WHERE ts.date BETWEEN ? AND ? ORDER BY ts.date, ts.conductor_score DESC"
		rows, err = db.Query(query, startDate, endDate)
	} else {
		query += " ORDER BY ts.date, ts.conductor_score DESC"
		rows, err = db.Query(query)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	schedules := []TrainSchedule{}
	for rows.Next() {
		var ts TrainSchedule
		var showedUp sql.NullBool
		var notes sql.NullString
		var score sql.NullInt64
		var actualConductorID sql.NullInt64
		var actualConductorName sql.NullString
		var vipID sql.NullInt64
		var vipName sql.NullString

		if err := rows.Scan(&ts.ID, &ts.Date, &ts.ConductorID, &ts.ConductorName,
			&score, &ts.BackupID, &ts.BackupName, &ts.BackupRank, &showedUp,
			&actualConductorID, &actualConductorName, &notes, &ts.CreatedAt,
			&vipID, &vipName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if showedUp.Valid {
			ts.ConductorShowedUp = &showedUp.Bool
		}
		if notes.Valid {
			ts.Notes = &notes.String
		}
		if score.Valid {
			scoreInt := int(score.Int64)
			ts.ConductorScore = &scoreInt
		}
		if actualConductorID.Valid {
			actualID := int(actualConductorID.Int64)
			ts.ActualConductorID = &actualID
		}
		if actualConductorName.Valid {
			ts.ActualConductorName = &actualConductorName.String
		}
		if vipID.Valid {
			v := int(vipID.Int64)
			ts.VipID = &v
		}
		if vipName.Valid && vipName.String != "" {
			ts.VipName = &vipName.String
		}

		schedules = append(schedules, ts)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedules)
}

// Create a train schedule
func createTrainSchedule(w http.ResponseWriter, r *http.Request) {
	var ts TrainSchedule
	if err := json.NewDecoder(r.Body).Decode(&ts); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate backup is R4 or R5
	var backupRank string
	err := db.QueryRow("SELECT rank FROM members WHERE id = ?", ts.BackupID).Scan(&backupRank)
	if err != nil {
		http.Error(w, "Backup member not found", http.StatusBadRequest)
		return
	}

	if backupRank != "R4" && backupRank != "R5" {
		http.Error(w, "Backup must be an R4 or R5 member", http.StatusBadRequest)
		return
	}

	// Fetch name snapshots (server-authoritative, not from client)
	var conductorSnapshot string
	db.QueryRow("SELECT name FROM members WHERE id = ?", ts.ConductorID).Scan(&conductorSnapshot)
	var backupSnapshot string
	db.QueryRow("SELECT name FROM members WHERE id = ?", ts.BackupID).Scan(&backupSnapshot)

	var vipIDVal interface{}
	var vipSnapshot string
	if ts.VipID != nil && *ts.VipID > 0 {
		vipIDVal = *ts.VipID
		db.QueryRow("SELECT name FROM members WHERE id = ?", *ts.VipID).Scan(&vipSnapshot)
	}

	// Use INSERT OR REPLACE to allow updating schedules created by auto-schedule
	result, err := db.Exec(
		"INSERT OR REPLACE INTO train_schedules (date, conductor_id, backup_id, conductor_score, conductor_showed_up, actual_conductor_id, notes, conductor_name_snapshot, backup_name_snapshot, vip_id, vip_name_snapshot) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		ts.Date, ts.ConductorID, ts.BackupID, ts.ConductorScore, ts.ConductorShowedUp, ts.ActualConductorID, ts.Notes, conductorSnapshot, backupSnapshot, vipIDVal, vipSnapshot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Awards and recommendations automatically become inactive via on-the-fly calculation

	id, _ := result.LastInsertId()
	ts.ID = int(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ts)
}

// Update a train schedule
func updateTrainSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	var ts TrainSchedule
	if err := json.NewDecoder(r.Body).Decode(&ts); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get existing schedule to check if conductor or backup changed
	var existingConductorID int
	var existingBackupID sql.NullInt64
	err = db.QueryRow("SELECT conductor_id, backup_id FROM train_schedules WHERE id = ?", id).Scan(&existingConductorID, &existingBackupID)
	if err != nil {
		http.Error(w, "Schedule not found", http.StatusNotFound)
		return
	}

	// Validate backup is R4 or R5 if backup is being updated
	if ts.BackupID > 0 {
		var backupRank string
		err := db.QueryRow("SELECT rank FROM members WHERE id = ?", ts.BackupID).Scan(&backupRank)
		if err != nil {
			http.Error(w, "Backup member not found", http.StatusBadRequest)
			return
		}

		if backupRank != "R4" && backupRank != "R5" {
			http.Error(w, "Backup must be an R4 or R5 member", http.StatusBadRequest)
			return
		}
	}

	// Fetch name snapshots for the updated schedule
	var updConductorSnapshot string
	db.QueryRow("SELECT name FROM members WHERE id = ?", ts.ConductorID).Scan(&updConductorSnapshot)
	var updBackupSnapshot string
	if ts.BackupID > 0 {
		db.QueryRow("SELECT name FROM members WHERE id = ?", ts.BackupID).Scan(&updBackupSnapshot)
	}

	var updVipIDVal interface{}
	var updVipSnapshot string
	if ts.VipID != nil && *ts.VipID > 0 {
		updVipIDVal = *ts.VipID
		db.QueryRow("SELECT name FROM members WHERE id = ?", *ts.VipID).Scan(&updVipSnapshot)
	}

	_, err = db.Exec(
		"UPDATE train_schedules SET date = ?, conductor_id = ?, backup_id = ?, conductor_score = ?, conductor_showed_up = ?, actual_conductor_id = ?, notes = ?, conductor_name_snapshot = ?, backup_name_snapshot = ?, vip_id = ?, vip_name_snapshot = ? WHERE id = ?",
		ts.Date, ts.ConductorID, ts.BackupID, ts.ConductorScore, ts.ConductorShowedUp, ts.ActualConductorID, ts.Notes, updConductorSnapshot, updBackupSnapshot, updVipIDVal, updVipSnapshot, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Awards and recommendations automatically become inactive via on-the-fly calculation

	ts.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ts)
}

// Delete a train schedule
func deleteTrainSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM train_schedules WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Auto-schedule train conductors and backups for a single day
func autoSchedule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		StartDate string `json:"start_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse the date and get the week start (Monday)
	scheduleDate, err := parseDate(input.StartDate)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	weekStart := getMondayOfWeek(scheduleDate)

	// Build ranking context
	ctx, err := buildRankingContext(weekStart)
	if err != nil {
		http.Error(w, "Failed to load ranking context: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get all eligible members
	rows, err := db.Query("SELECT id, name, rank, COALESCE(eligible, 1) FROM members WHERE COALESCE(eligible, 1) = 1 AND deleted_at IS NULL ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var candidates []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Name, &m.Rank, &m.Eligible); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		candidates = append(candidates, m)
	}

	if len(candidates) < 7 {
		http.Error(w, "Not enough members for weekly scheduling (need at least 7)", http.StatusBadRequest)
		return
	}

	// Score each candidate using the abstracted ranking system
	type ScoredMember struct {
		Member Member
		Score  int
	}

	var scoredCandidates []ScoredMember
	for _, member := range candidates {
		score := calculateMemberScore(member, ctx)
		scoredCandidates = append(scoredCandidates, ScoredMember{
			Member: member,
			Score:  score,
		})
	}

	// Sort by score (highest first)
	for i := 0; i < len(scoredCandidates); i++ {
		for j := i + 1; j < len(scoredCandidates); j++ {
			if scoredCandidates[j].Score > scoredCandidates[i].Score {
				scoredCandidates[i], scoredCandidates[j] = scoredCandidates[j], scoredCandidates[i]
			}
		}
	}

	// Pre-select top 7 performers as conductors for the week
	plannedConductors := make(map[int]bool)
	for i := 0; i < 7 && i < len(scoredCandidates); i++ {
		plannedConductors[scoredCandidates[i].Member.ID] = true
	}

	// Now schedule each day
	var weekSchedules []TrainSchedule
	usedConductors := make(map[int]bool)
	usedBackups := make(map[int]bool)

	for day := 0; day < 7; day++ {
		currentDate := weekStart.AddDate(0, 0, day)
		dateStr := formatDateString(currentDate)

		// Select conductor from top 7 who hasn't been assigned yet
		var conductorID int
		var conductorScore int

		for _, sc := range scoredCandidates {
			if plannedConductors[sc.Member.ID] && !usedConductors[sc.Member.ID] {
				conductorID = sc.Member.ID
				conductorScore = sc.Score
				usedConductors[sc.Member.ID] = true
				break
			}
		}

		if conductorID == 0 {
			http.Error(w, "Unable to assign conductor for all days", http.StatusInternalServerError)
			return
		}

		// Select backup (must be R4/R5, not a planned conductor, not already used as backup)
		var availableBackups []Member
		for _, sc := range scoredCandidates {
			if !plannedConductors[sc.Member.ID] &&
				!usedBackups[sc.Member.ID] &&
				(sc.Member.Rank == "R4" || sc.Member.Rank == "R5") {
				availableBackups = append(availableBackups, sc.Member)
			}
		}

		var backupID int
		var backupMemberName string
		if len(availableBackups) > 0 {
			// Randomly select from available backups
			randomIndex := time.Now().UnixNano() % int64(len(availableBackups))
			backupID = availableBackups[randomIndex].ID
			backupMemberName = availableBackups[randomIndex].Name
			usedBackups[backupID] = true
		}

		// Find conductor name for snapshot
		var autoCondName string
		for _, sc := range scoredCandidates {
			if sc.Member.ID == conductorID {
				autoCondName = sc.Member.Name
				break
			}
		}

		// If backupID is 0, no backup available - continue anyway and allow manual assignment

		// Insert schedule for this day
		var result sql.Result
		if backupID > 0 {
			result, err = db.Exec(
				"INSERT OR REPLACE INTO train_schedules (date, conductor_id, backup_id, conductor_score, conductor_name_snapshot, backup_name_snapshot) VALUES (?, ?, ?, ?, ?, ?)",
				dateStr, conductorID, backupID, conductorScore, autoCondName, backupMemberName,
			)
		} else {
			result, err = db.Exec(
				"INSERT OR REPLACE INTO train_schedules (date, conductor_id, backup_id, conductor_score, conductor_name_snapshot) VALUES (?, ?, NULL, ?, ?)",
				dateStr, conductorID, conductorScore, autoCondName,
			)
		}
		if err != nil {
			http.Error(w, "Failed to create schedule: "+err.Error(), http.StatusInternalServerError)
			return
		}

		scheduleID, _ := result.LastInsertId()

		// Get the full schedule details
		var schedule TrainSchedule
		var score sql.NullInt64
		var backupName sql.NullString
		var backupRank sql.NullString

		if backupID > 0 {
			err = db.QueryRow(`
			SELECT 
				ts.id, ts.date, ts.conductor_id, 
				mc.name, ts.conductor_score, ts.backup_id, mb.name, mb.rank,
				ts.conductor_showed_up, ts.notes, ts.created_at
			FROM train_schedules ts
			JOIN members mc ON ts.conductor_id = mc.id
			LEFT JOIN members mb ON ts.backup_id = mb.id
			WHERE ts.id = ?
		`, scheduleID).Scan(
				&schedule.ID, &schedule.Date, &schedule.ConductorID,
				&schedule.ConductorName, &score, &schedule.BackupID, &backupName,
				&backupRank, &schedule.ConductorShowedUp, &schedule.Notes,
				&schedule.CreatedAt,
			)
		} else {
			err = db.QueryRow(`
			SELECT 
				ts.id, ts.date, ts.conductor_id, 
				mc.name, ts.conductor_score,
				ts.conductor_showed_up, ts.notes, ts.created_at
			FROM train_schedules ts
			JOIN members mc ON ts.conductor_id = mc.id
			WHERE ts.id = ?
		`, scheduleID).Scan(
				&schedule.ID, &schedule.Date, &schedule.ConductorID,
				&schedule.ConductorName, &score,
				&schedule.ConductorShowedUp, &schedule.Notes,
				&schedule.CreatedAt,
			)
			// Set backup fields to empty/zero
			schedule.BackupID = 0
			schedule.BackupName = ""
			schedule.BackupRank = ""
		}

		if err != nil {
			http.Error(w, "Failed to retrieve schedule: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if backupID > 0 {
			if backupName.Valid {
				schedule.BackupName = backupName.String
			}
			if backupRank.Valid {
				schedule.BackupRank = backupRank.String
			}
		}

		if score.Valid {
			scoreInt := int(score.Int64)
			schedule.ConductorScore = &scoreInt
		}

		weekSchedules = append(weekSchedules, schedule)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Week scheduled successfully",
		"schedules": weekSchedules,
	})
}

// Helper function to get server location
func getServerLocation() *time.Location {
	settings, err := loadSettings()
	if err != nil || settings.ServerTimezone == "" {
		// Default to UTC if settings can't be loaded
		loc, _ := time.LoadLocation("UTC")
		return loc
	}

	loc, err := time.LoadLocation(settings.ServerTimezone)
	if err != nil {
		// Fall back to UTC if timezone is invalid
		log.Printf("Warning: Invalid timezone '%s', falling back to UTC", settings.ServerTimezone)
		loc, _ = time.LoadLocation("UTC")
		return loc
	}

	return loc
}

// Helper function to get current time in game/alliance timezone
func getServerTime() time.Time {
	return time.Now().In(getServerLocation())
}

// Helper function to parse date string in game/alliance timezone
func parseDate(dateStr string) (time.Time, error) {
	loc := getServerLocation()
	return time.ParseInLocation("2006-01-02", dateStr, loc)
}

// Helper function to format date to string
func formatDateString(t time.Time) string {
	return t.In(getServerLocation()).Format("2006-01-02")
}

// Helper function to get Monday of a week
func getMondayOfWeek(date time.Time) time.Time {
	offset := int(time.Monday - date.Weekday())
	if offset > 0 {
		offset = -6
	}
	return date.AddDate(0, 0, offset)
}

// Helper function to format time across multiple timezones with DST support
// baseTime is in HH:MM format (e.g., "15:00")
// date is the reference date for DST calculation
// Returns formatted string like "15:00 ST / 17:00 GMT / 18:00 CET"
func formatTimeAcrossTimezones(baseTime string, date time.Time, settings Settings) string {
	// Parse base time
	timeParts := strings.Split(baseTime, ":")
	if len(timeParts) != 2 {
		return baseTime // Invalid format, return as-is
	}

	hour, err1 := strconv.Atoi(timeParts[0])
	minute, err2 := strconv.Atoi(timeParts[1])
	if err1 != nil || err2 != nil {
		return baseTime // Invalid format, return as-is
	}

	// Create a time in the game timezone on the given date
	gameLoc := getServerLocation()
	gameTime := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, gameLoc)

	// Parse display timezones from JSON
	var displayTimezones []string
	if err := json.Unmarshal([]byte(settings.DisplayTimezones), &displayTimezones); err != nil {
		// If parsing fails, use default
		displayTimezones = []string{"Europe/London"}
	}

	// Build formatted string
	var parts []string

	// Add server time (ST) first
	parts = append(parts, fmt.Sprintf("%s ST", baseTime))

	// Add each display timezone
	for _, tzName := range displayTimezones {
		loc, err := time.LoadLocation(tzName)
		if err != nil {
			continue // Skip invalid timezones
		}

		// Convert game time to this timezone
		timeInTz := gameTime.In(loc)

		// Get timezone abbreviation (handles DST automatically)
		tzAbbr := timeInTz.Format("MST")

		// Format time
		parts = append(parts, fmt.Sprintf("%02d:%02d %s", timeInTz.Hour(), timeInTz.Minute(), tzAbbr))
	}

	return strings.Join(parts, " / ")
}

// Import members from CSV
func importCSV(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (10MB max)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, "Failed to parse CSV: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(records) == 0 {
		http.Error(w, "CSV file is empty", http.StatusBadRequest)
		return
	}

	// Skip header row if it looks like a header
	startIndex := 0
	if len(records) > 0 {
		firstRow := records[0]
		if len(firstRow) > 0 {
			firstCell := strings.ToLower(strings.TrimSpace(firstRow[0]))
			// Check if first row is a header
			if firstCell == "username" || firstCell == "name" || firstCell == "member" {
				startIndex = 1
			}
		}
	}

	validRanks := map[string]bool{"R1": true, "R2": true, "R3": true, "R4": true, "R5": true}
	detectedMembers := []DetectedMember{}
	errors := []string{}

	// Get existing active members
	existingMembers := make(map[string]Member)
	rows, err := db.Query("SELECT id, name, rank FROM members WHERE deleted_at IS NULL")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var m Member
			rows.Scan(&m.ID, &m.Name, &m.Rank)
			existingMembers[m.Name] = m
		}
	}

	for i := startIndex; i < len(records); i++ {
		record := records[i]

		// New format: Username,Rank,Power,Level,Status,Last_Active
		// We only care about Username (index 0) and Rank (index 1)
		if len(record) < 2 {
			errors = append(errors, fmt.Sprintf("Line %d: Insufficient columns (need at least Username and Rank)", i+1))
			continue
		}

		name := strings.TrimSpace(record[0])
		rank := strings.TrimSpace(record[1])

		if name == "" {
			errors = append(errors, fmt.Sprintf("Line %d: Empty username", i+1))
			continue
		}

		if !validRanks[rank] {
			errors = append(errors, fmt.Sprintf("Line %d: Invalid rank '%s' (must be R1-R5)", i+1, rank))
			continue
		}

		detected := DetectedMember{
			Name: name,
			Rank: rank,
		}

		// Check if member exists
		if existing, found := existingMembers[name]; found {
			// Existing member - check for rank change
			if existing.Rank != rank {
				detected.RankChanged = true
				detected.OldRank = existing.Rank
			}
		} else {
			// New member - check for similar names in existing members
			detected.IsNew = true
			similarNames := []string{}
			for existingName := range existingMembers {
				if areSimilar(name, existingName) {
					similarNames = append(similarNames, existingName)
				}
			}
			if len(similarNames) > 0 {
				detected.SimilarMatch = similarNames
			}
		}

		detectedMembers = append(detectedMembers, detected)
	}

	// Find members that would be removed (in database but not in CSV)
	membersToRemove := []MemberToRemove{}
	csvNames := make(map[string]bool)
	for _, m := range detectedMembers {
		csvNames[m.Name] = true
	}
	for _, existing := range existingMembers {
		if !csvNames[existing.Name] {
			membersToRemove = append(membersToRemove, MemberToRemove{
				ID:   existing.ID,
				Name: existing.Name,
				Rank: existing.Rank,
			})
		}
	}

	// Return preview data
	result := map[string]interface{}{
		"detected_members":  detectedMembers,
		"members_to_remove": membersToRemove,
		"errors":            errors,
		"total_rows":        len(records) - startIndex,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Confirm CSV import (reuses the same confirmMemberUpdates function)
// The route will be /api/members/import/confirm

// Get awards for a specific week or all weeks
func getAwards(w http.ResponseWriter, r *http.Request) {
	weekDate := r.URL.Query().Get("week")

	var query string
	var rows *sql.Rows
	var err error

	if weekDate != "" {
		query = `
			SELECT a.id, a.week_date, a.award_type, a.rank, a.member_id,
			       COALESCE(m.name, a.member_name_snapshot, '[Removed Member]'), m.nickname, a.created_at
			FROM awards a
			LEFT JOIN members m ON a.member_id = m.id AND m.deleted_at IS NULL
			WHERE a.week_date = ?
			ORDER BY a.award_type, a.rank
		`
		rows, err = db.Query(query, weekDate)
	} else {
		query = `
			SELECT a.id, a.week_date, a.award_type, a.rank, a.member_id,
			       COALESCE(m.name, a.member_name_snapshot, '[Removed Member]'), m.nickname, a.created_at
			FROM awards a
			LEFT JOIN members m ON a.member_id = m.id AND m.deleted_at IS NULL
			ORDER BY a.week_date DESC, a.award_type, a.rank
		`
		rows, err = db.Query(query)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	awards := []Award{}
	for rows.Next() {
		var a Award
		var nick sql.NullString
		if err := rows.Scan(&a.ID, &a.WeekDate, &a.AwardType, &a.Rank, &a.MemberID, &a.MemberName, &nick, &a.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if nick.Valid && nick.String != "" {
			a.MemberNickname = &nick.String
		}
		awards = append(awards, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(awards)
}

// Save awards for a week (bulk operation)
func saveAwards(w http.ResponseWriter, r *http.Request) {
	var data struct {
		WeekDate string `json:"week_date"`
		Awards   []struct {
			AwardType string `json:"award_type"`
			Rank      int    `json:"rank"`
			MemberID  int    `json:"member_id"`
		} `json:"awards"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Delete existing awards for this week
	_, err = tx.Exec("DELETE FROM awards WHERE week_date = ?", data.WeekDate)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to clear existing awards", http.StatusInternalServerError)
		return
	}

	// Insert new awards
	for _, award := range data.Awards {
		if award.MemberID > 0 { // Only insert if a member is selected
			var awardMemberName string
			tx.QueryRow("SELECT name FROM members WHERE id = ?", award.MemberID).Scan(&awardMemberName)
			_, err = tx.Exec(
				"INSERT INTO awards (week_date, award_type, rank, member_id, member_name_snapshot) VALUES (?, ?, ?, ?, ?)",
				data.WeekDate, award.AwardType, award.Rank, award.MemberID, awardMemberName)
			if err != nil {
				tx.Rollback()
				http.Error(w, "Failed to save award", http.StatusInternalServerError)
				return
			}
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		http.Error(w, "Failed to save changes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Awards saved successfully"})
}

// Delete awards for a specific week
func deleteWeekAwards(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	weekDate := vars["week"]

	_, err := db.Exec("DELETE FROM awards WHERE week_date = ?", weekDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Get VS points for a week or all weeks
func getVSPoints(w http.ResponseWriter, r *http.Request) {
	weekDate := r.URL.Query().Get("week")

	var query string
	var rows *sql.Rows
	var err error

	if weekDate != "" {
		query = `
			SELECT v.id, v.member_id, v.week_date, v.monday, v.tuesday, v.wednesday, 
			       v.thursday, v.friday, v.saturday, v.created_at, v.updated_at,
			       COALESCE(m.name, '[Inactive]'), COALESCE(m.rank, '')
			FROM vs_points v
			LEFT JOIN members m ON v.member_id = m.id AND m.deleted_at IS NULL
			WHERE v.week_date = ?
			ORDER BY COALESCE(m.name, '[Inactive]')
		`
		rows, err = db.Query(query, weekDate)
	} else {
		query = `
			SELECT v.id, v.member_id, v.week_date, v.monday, v.tuesday, v.wednesday, 
			       v.thursday, v.friday, v.saturday, v.created_at, v.updated_at,
			       COALESCE(m.name, '[Inactive]'), COALESCE(m.rank, '')
			FROM vs_points v
			LEFT JOIN members m ON v.member_id = m.id AND m.deleted_at IS NULL
			ORDER BY v.week_date DESC, COALESCE(m.name, '[Inactive]')
		`
		rows, err = db.Query(query)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	vsPoints := []VSPointsWithMember{}
	for rows.Next() {
		var v VSPointsWithMember
		if err := rows.Scan(&v.ID, &v.MemberID, &v.WeekDate, &v.Monday, &v.Tuesday,
			&v.Wednesday, &v.Thursday, &v.Friday, &v.Saturday, &v.CreatedAt, &v.UpdatedAt,
			&v.MemberName, &v.MemberRank); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vsPoints = append(vsPoints, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vsPoints)
}

// Save VS points for a week (bulk operation)
func saveVSPoints(w http.ResponseWriter, r *http.Request) {
	var data struct {
		WeekDate string `json:"week_date"`
		Points   []struct {
			MemberID  int `json:"member_id"`
			Monday    int `json:"monday"`
			Tuesday   int `json:"tuesday"`
			Wednesday int `json:"wednesday"`
			Thursday  int `json:"thursday"`
			Friday    int `json:"friday"`
			Saturday  int `json:"saturday"`
		} `json:"points"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Upsert VS points for each member
	for _, point := range data.Points {
		// Check if record exists
		var existingID int
		err = tx.QueryRow("SELECT id FROM vs_points WHERE member_id = ? AND week_date = ?",
			point.MemberID, data.WeekDate).Scan(&existingID)

		if err == sql.ErrNoRows {
			// Insert new record
			_, err = tx.Exec(`
				INSERT INTO vs_points (member_id, week_date, monday, tuesday, wednesday, thursday, friday, saturday, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
				point.MemberID, data.WeekDate, point.Monday, point.Tuesday, point.Wednesday,
				point.Thursday, point.Friday, point.Saturday)
		} else if err == nil {
			// Update existing record
			_, err = tx.Exec(`
				UPDATE vs_points 
				SET monday = ?, tuesday = ?, wednesday = ?, thursday = ?, friday = ?, saturday = ?, updated_at = CURRENT_TIMESTAMP
				WHERE member_id = ? AND week_date = ?`,
				point.Monday, point.Tuesday, point.Wednesday, point.Thursday, point.Friday, point.Saturday,
				point.MemberID, data.WeekDate)
		}

		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to save VS points", http.StatusInternalServerError)
			return
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		http.Error(w, "Failed to save changes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "VS points saved successfully"})
}

// Patch a single member's VS points for one day (R4/R5/Admin only)
func patchVSPoint(w http.ResponseWriter, r *http.Request) {
	var data struct {
		WeekDate string `json:"week_date"`
		MemberID int    `json:"member_id"`
		Day      string `json:"day"`
		Value    int    `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if data.Value < 0 {
		data.Value = 0
	}

	// Build safe upsert query using validated column name
	var query string
	switch data.Day {
	case "monday":
		query = `INSERT INTO vs_points (member_id, week_date, monday) VALUES (?,?,?) ON CONFLICT(member_id,week_date) DO UPDATE SET monday=?`
	case "tuesday":
		query = `INSERT INTO vs_points (member_id, week_date, tuesday) VALUES (?,?,?) ON CONFLICT(member_id,week_date) DO UPDATE SET tuesday=?`
	case "wednesday":
		query = `INSERT INTO vs_points (member_id, week_date, wednesday) VALUES (?,?,?) ON CONFLICT(member_id,week_date) DO UPDATE SET wednesday=?`
	case "thursday":
		query = `INSERT INTO vs_points (member_id, week_date, thursday) VALUES (?,?,?) ON CONFLICT(member_id,week_date) DO UPDATE SET thursday=?`
	case "friday":
		query = `INSERT INTO vs_points (member_id, week_date, friday) VALUES (?,?,?) ON CONFLICT(member_id,week_date) DO UPDATE SET friday=?`
	case "saturday":
		query = `INSERT INTO vs_points (member_id, week_date, saturday) VALUES (?,?,?) ON CONFLICT(member_id,week_date) DO UPDATE SET saturday=?`
	default:
		http.Error(w, "Invalid day — must be monday–saturday", http.StatusBadRequest)
		return
	}

	if _, err := db.Exec(query, data.MemberID, data.WeekDate, data.Value, data.Value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "updated"})
}

// Delete VS points for a specific week
func deleteWeekVSPoints(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	weekDate := vars["week"]

	_, err := db.Exec("DELETE FROM vs_points WHERE week_date = ?", weekDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// VS compliance report: per-member daily/weekly target status for the last N weeks
func getVSCompliance(w http.ResponseWriter, r *http.Request) {
	weeksParam := r.URL.Query().Get("weeks")
	numWeeks := 4
	if weeksParam != "" {
		if n, err := strconv.Atoi(weeksParam); err == nil && n > 0 && n <= 12 {
			numWeeks = n
		}
	}

	// Load targets from settings
	var dailyTarget, weeklyTarget int
	db.QueryRow(`SELECT COALESCE(vs_points_daily_target,0), COALESCE(vs_points_weekly_target,0) FROM settings WHERE id=1`).Scan(&dailyTarget, &weeklyTarget)

	// Load all active members
	memberRows, err := db.Query(`SELECT id, name, COALESCE(nickname,''), rank FROM members WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer memberRows.Close()

	type Member struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Nickname string `json:"nickname,omitempty"`
		Rank     string `json:"rank"`
	}
	var members []Member
	for memberRows.Next() {
		var m Member
		if err := memberRows.Scan(&m.ID, &m.Name, &m.Nickname, &m.Rank); err != nil {
			continue
		}
		members = append(members, m)
	}
	memberRows.Close()

	type MemberCompliance struct {
		MemberID    int    `json:"member_id"`
		Name        string `json:"name"`
		Nickname    string `json:"nickname,omitempty"`
		Rank        string `json:"rank"`
		Monday      int    `json:"monday"`
		Tuesday     int    `json:"tuesday"`
		Wednesday   int    `json:"wednesday"`
		Thursday    int    `json:"thursday"`
		Friday      int    `json:"friday"`
		Saturday    int    `json:"saturday"`
		WeeklyTotal int    `json:"weekly_total"`
		WeeklyMet   bool   `json:"weekly_met"`
		DaysMet     int    `json:"days_met"`
		HasData     bool   `json:"has_data"`
	}

	type WeekCompliance struct {
		WeekDate  string             `json:"week_date"`
		WeekLabel string             `json:"week_label"`
		Members   []MemberCompliance `json:"members"`
	}

	now := getServerTime()
	currentMonday := getMondayOfWeek(now)

	// Build vs_points lookup for all needed weeks in one query
	startDate := currentMonday.AddDate(0, 0, -(numWeeks-1)*7)
	vsRows, err := db.Query(`
		SELECT member_id, week_date,
		       COALESCE(monday,0), COALESCE(tuesday,0), COALESCE(wednesday,0),
		       COALESCE(thursday,0), COALESCE(friday,0), COALESCE(saturday,0)
		FROM vs_points
		WHERE week_date >= ?
		ORDER BY week_date ASC
	`, formatDateString(startDate))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer vsRows.Close()

	// Map: weekDate -> memberID -> row data
	type vsRow struct{ Mon, Tue, Wed, Thu, Fri, Sat int }
	vsData := make(map[string]map[int]vsRow)
	for vsRows.Next() {
		var memberID int
		var weekDate string
		var row vsRow
		if err := vsRows.Scan(&memberID, &weekDate, &row.Mon, &row.Tue, &row.Wed, &row.Thu, &row.Fri, &row.Sat); err != nil {
			continue
		}
		if vsData[weekDate] == nil {
			vsData[weekDate] = make(map[int]vsRow)
		}
		vsData[weekDate][memberID] = row
	}
	vsRows.Close()

	// Build compliance result for each week (most recent first)
	weeks := []WeekCompliance{}
	for i := numWeeks - 1; i >= 0; i-- {
		monday := currentMonday.AddDate(0, 0, -i*7)
		sunday := monday.AddDate(0, 0, 6)
		weekDate := formatDateString(monday)
		weekLabel := fmt.Sprintf("%s – %s", monday.Format("Jan 2"), sunday.Format("Jan 2"))

		weekVS := vsData[weekDate]
		memberList := make([]MemberCompliance, 0, len(members))
		for _, m := range members {
			mc := MemberCompliance{MemberID: m.ID, Name: m.Name, Nickname: m.Nickname, Rank: m.Rank}
			if row, ok := weekVS[m.ID]; ok {
				mc.Monday, mc.Tuesday, mc.Wednesday = row.Mon, row.Tue, row.Wed
				mc.Thursday, mc.Friday, mc.Saturday = row.Thu, row.Fri, row.Sat
				mc.WeeklyTotal = row.Mon + row.Tue + row.Wed + row.Thu + row.Fri + row.Sat
				mc.HasData = true
				days := []int{row.Mon, row.Tue, row.Wed, row.Thu, row.Fri, row.Sat}
				for _, d := range days {
					if dailyTarget > 0 && d >= dailyTarget {
						mc.DaysMet++
					} else if dailyTarget == 0 && d > 0 {
						mc.DaysMet++
					}
				}
				mc.WeeklyMet = weeklyTarget > 0 && mc.WeeklyTotal >= weeklyTarget
			}
			memberList = append(memberList, mc)
		}
		weeks = append(weeks, WeekCompliance{WeekDate: weekDate, WeekLabel: weekLabel, Members: memberList})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"targets": map[string]int{
			"daily":  dailyTarget,
			"weekly": weeklyTarget,
		},
		"weeks": weeks,
	})
}

// Get all award types
func getAwardTypes(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, active, sort_order, created_at
		FROM award_types
		ORDER BY sort_order, name
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	awardTypes := []AwardType{}
	for rows.Next() {
		var at AwardType
		if err := rows.Scan(&at.ID, &at.Name, &at.Active, &at.SortOrder, &at.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		awardTypes = append(awardTypes, at)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(awardTypes)
}

// Create a new award type
func createAwardType(w http.ResponseWriter, r *http.Request) {
	var at AwardType
	if err := json.NewDecoder(r.Body).Decode(&at); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate name
	if strings.TrimSpace(at.Name) == "" {
		http.Error(w, "Award type name is required", http.StatusBadRequest)
		return
	}

	// Check if award type already exists
	var existingID int
	err := db.QueryRow("SELECT id FROM award_types WHERE name = ?", at.Name).Scan(&existingID)
	if err == nil {
		http.Error(w, "Award type already exists", http.StatusConflict)
		return
	}

	// Get max sort_order and add 1
	var maxOrder int
	err = db.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM award_types").Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	result, err := db.Exec(
		"INSERT INTO award_types (name, active, sort_order) VALUES (?, ?, ?)",
		at.Name, true, maxOrder+1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	at.ID = int(id)
	at.Active = true
	at.SortOrder = maxOrder + 1
	at.CreatedAt = time.Now().Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(at)
}

// Update award type (mainly for active status)
func updateAwardType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var at AwardType
	if err := json.NewDecoder(r.Body).Decode(&at); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec(
		"UPDATE award_types SET active = ?, name = ? WHERE id = ?",
		at.Active, at.Name, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Award type updated"})
}

// Delete award type (supports force deletion)
func deleteAwardType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	// Get award type name
	var name string
	err = db.QueryRow("SELECT name FROM award_types WHERE id = ?", id).Scan(&name)
	if err != nil {
		http.Error(w, "Award type not found", http.StatusNotFound)
		return
	}

	if force {
		// Delete all awards of this type first
		_, err = db.Exec("DELETE FROM awards WHERE award_type = ?", name)
		if err != nil {
			http.Error(w, "Failed to delete related awards", http.StatusInternalServerError)
			return
		}
	} else {
		// Check if award type is used in any awards
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM awards WHERE award_type = ?", name).Scan(&count)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if count > 0 {
			http.Error(w, "Cannot delete award type that is used in awards. Use force=true to delete anyway.", http.StatusBadRequest)
			return
		}
	}

	_, err = db.Exec("DELETE FROM award_types WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Get all recommendations
func getRecommendations(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT 
			rec.id, 
			rec.member_id, 
			COALESCE(m.name, rec.member_name_snapshot, '[Removed Member]'),
			COALESCE(m.rank, ''),
			m.nickname,
			u.username,
			rec.recommended_by_id,
			COALESCE(rec.notes, ''),
			rec.created_at,
			CASE 
				WHEN EXISTS (
					SELECT 1 FROM train_schedules ts
					WHERE (ts.conductor_id = rec.member_id 
					       OR (ts.backup_id = rec.member_id AND ts.conductor_showed_up = 0 AND ts.actual_conductor_id IS NULL)
					       OR ts.actual_conductor_id = rec.member_id)
					AND ts.date >= date(rec.created_at)
				) THEN 1
				ELSE 0
			END as expired
		FROM recommendations rec
		LEFT JOIN members m ON rec.member_id = m.id AND m.deleted_at IS NULL
		JOIN users u ON rec.recommended_by_id = u.id
		ORDER BY rec.created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	recommendations := []Recommendation{}
	for rows.Next() {
		var rec Recommendation
		var nick sql.NullString
		if err := rows.Scan(&rec.ID, &rec.MemberID, &rec.MemberName, &rec.MemberRank,
			&nick, &rec.RecommendedBy, &rec.RecommendedByID, &rec.Notes, &rec.CreatedAt, &rec.Expired); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if nick.Valid && nick.String != "" {
			rec.MemberNickname = &nick.String
		}
		recommendations = append(recommendations, rec)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recommendations)
}

// Create recommendation
func createRecommendation(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID := session.Values["user_id"].(int)

	var input struct {
		MemberID int    `json:"member_id"`
		Notes    string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.MemberID == 0 {
		http.Error(w, "Member ID is required", http.StatusBadRequest)
		return
	}

	// Check if member exists and is active
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM members WHERE id = ? AND deleted_at IS NULL)", input.MemberID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	// Get member name for snapshot
	var recMemberName string
	db.QueryRow("SELECT name FROM members WHERE id = ?", input.MemberID).Scan(&recMemberName)

	result, err := db.Exec(
		"INSERT INTO recommendations (member_id, recommended_by_id, notes, member_name_snapshot) VALUES (?, ?, ?, ?)",
		input.MemberID, userID, input.Notes, recMemberName,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	// Get the created recommendation
	var rec Recommendation
	err = db.QueryRow(`
		SELECT 
			rec.id, 
			rec.member_id, 
			m.name, 
			m.rank,
			u.username,
			rec.recommended_by_id,
			COALESCE(rec.notes, ''),
			rec.created_at,
			CASE 
				WHEN EXISTS (
					SELECT 1 FROM train_schedules ts
					WHERE (ts.conductor_id = rec.member_id 
					       OR (ts.backup_id = rec.member_id AND ts.conductor_showed_up = 0 AND ts.actual_conductor_id IS NULL)
					       OR ts.actual_conductor_id = rec.member_id)
					AND ts.date >= date(rec.created_at)
				) THEN 1
				ELSE 0
			END as expired
		FROM recommendations rec
		LEFT JOIN members m ON rec.member_id = m.id AND m.deleted_at IS NULL
		JOIN users u ON rec.recommended_by_id = u.id
		WHERE rec.id = ?
	`, id).Scan(&rec.ID, &rec.MemberID, &rec.MemberName, &rec.MemberRank,
		&rec.RecommendedBy, &rec.RecommendedByID, &rec.Notes, &rec.CreatedAt, &rec.Expired)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rec)
}

// Delete recommendation
func deleteRecommendation(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID := session.Values["user_id"].(int)
	isAdmin := session.Values["is_admin"].(bool)

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Check if user is the one who created the recommendation or is admin
	var recommendedByID int
	err = db.QueryRow("SELECT recommended_by_id FROM recommendations WHERE id = ?", id).Scan(&recommendedByID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Recommendation not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if recommendedByID != userID && !isAdmin {
		http.Error(w, "You can only delete your own recommendations", http.StatusForbidden)
		return
	}

	_, err = db.Exec("DELETE FROM recommendations WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Get conduct reports (expires after 1 week)
func getConductReports(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT 
			dr.id, 
			dr.member_id, 
			COALESCE(m.name, dr.member_name_snapshot, '[Removed Member]'),
			COALESCE(m.rank, ''),
			m.nickname,
			dr.points,
			dr.notes,
			u.username,
			dr.created_by_id,
			dr.created_at,
			CASE 
				WHEN datetime(dr.created_at, '+7 days') < datetime('now') THEN 1
				ELSE 0
			END as expired
		FROM dyno_recommendations dr
		LEFT JOIN members m ON dr.member_id = m.id AND m.deleted_at IS NULL
		JOIN users u ON dr.created_by_id = u.id
		ORDER BY dr.created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	conductReports := []ConductReport{}
	for rows.Next() {
		var dr ConductReport
		var nick sql.NullString
		if err := rows.Scan(&dr.ID, &dr.MemberID, &dr.MemberName, &dr.MemberRank,
			&nick, &dr.Points, &dr.Notes, &dr.CreatedBy, &dr.CreatedByID, &dr.CreatedAt, &dr.Expired); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if nick.Valid && nick.String != "" {
			dr.MemberNickname = &nick.String
		}
		conductReports = append(conductReports, dr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conductReports)
}

// Create conduct report
func createConductReport(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID := session.Values["user_id"].(int)

	var input struct {
		MemberID int    `json:"member_id"`
		Points   int    `json:"points"`
		Notes    string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.MemberID == 0 {
		http.Error(w, "Member ID is required", http.StatusBadRequest)
		return
	}

	if input.Notes == "" {
		http.Error(w, "Notes are required", http.StatusBadRequest)
		return
	}

	// Check if member exists and is active
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM members WHERE id = ? AND deleted_at IS NULL)", input.MemberID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	// Get member name for snapshot
	var conductMemberName string
	db.QueryRow("SELECT name FROM members WHERE id = ?", input.MemberID).Scan(&conductMemberName)

	result, err := db.Exec(
		"INSERT INTO dyno_recommendations (member_id, points, notes, created_by_id, member_name_snapshot) VALUES (?, ?, ?, ?, ?)",
		input.MemberID, input.Points, input.Notes, userID, conductMemberName,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	// Get the created conduct report
	var dr ConductReport
	err = db.QueryRow(`
		SELECT 
			dr.id, 
			dr.member_id, 
			m.name, 
			m.rank,
			dr.points,
			dr.notes,
			u.username,
			dr.created_by_id,
			dr.created_at,
			CASE 
				WHEN datetime(dr.created_at, '+7 days') < datetime('now') THEN 1
				ELSE 0
			END as expired
		FROM dyno_recommendations dr
		LEFT JOIN members m ON dr.member_id = m.id AND m.deleted_at IS NULL
		JOIN users u ON dr.created_by_id = u.id
		WHERE dr.id = ?
	`, id).Scan(&dr.ID, &dr.MemberID, &dr.MemberName, &dr.MemberRank,
		&dr.Points, &dr.Notes, &dr.CreatedBy, &dr.CreatedByID, &dr.CreatedAt, &dr.Expired)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dr)
}

// Delete conduct report
func deleteConductReport(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID := session.Values["user_id"].(int)
	isAdmin := session.Values["is_admin"].(bool)

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Check if user is the one who created the recommendation or is admin
	var createdByID int
	err = db.QueryRow("SELECT created_by_id FROM dyno_recommendations WHERE id = ?", id).Scan(&createdByID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Conduct report not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if createdByID != userID && !isAdmin {
		http.Error(w, "You can only delete your own conduct reports", http.StatusForbidden)
		return
	}

	_, err = db.Exec("DELETE FROM dyno_recommendations WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Get settings
func getSettings(w http.ResponseWriter, r *http.Request) {
	var settings Settings
	err := db.QueryRow(`SELECT id, 
		COALESCE(alliance_name, 'Last War: Survival') as alliance_name,
		COALESCE(alliance_short_name, 'LWS') as alliance_short_name,
		award_first_points, award_second_points, award_third_points, 
		recommendation_points, recent_conductor_penalty_days, above_average_conductor_penalty, r4r5_rank_boost,
		first_time_conductor_boost, schedule_message_template, daily_message_template, 
		COALESCE(power_tracking_enabled, 0) as power_tracking_enabled,
		COALESCE(server_timezone, 'UTC') as server_timezone,
		COALESCE(conductor_time, '15:00') as conductor_time,
		COALESCE(backup_time, '16:30') as backup_time,
		COALESCE(display_timezones, '["Europe/London"]') as display_timezones,
		COALESCE(vs_points_daily_target, 0) as vs_points_daily_target,
		COALESCE(vs_points_weekly_target, 0) as vs_points_weekly_target,
		COALESCE(min_power, 0) as min_power,
		COALESCE(min_hq_level, 0) as min_hq_level,
		COALESCE(vip_seat_enabled, 1) as vip_seat_enabled,
		COALESCE(marshal_guard_enabled, 1) as marshal_guard_enabled
		FROM settings WHERE id = 1`).Scan(
		&settings.ID,
		&settings.AllianceName,
		&settings.AllianceShortName,
		&settings.AwardFirstPoints,
		&settings.AwardSecondPoints,
		&settings.AwardThirdPoints,
		&settings.RecommendationPoints,
		&settings.RecentConductorPenaltyDays,
		&settings.AboveAverageConductorPenalty,
		&settings.R4R5RankBoost,
		&settings.FirstTimeConductorBoost,
		&settings.ScheduleMessageTemplate,
		&settings.DailyMessageTemplate,
		&settings.PowerTrackingEnabled,
		&settings.ServerTimezone,
		&settings.ConductorTime,
		&settings.BackupTime,
		&settings.DisplayTimezones,
		&settings.VSPointsDailyTarget,
		&settings.VSPointsWeeklyTarget,
		&settings.MinPower,
		&settings.MinHQLevel,
		&settings.VipSeatEnabled,
		&settings.MarshalGuardEnabled,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// Update settings (admin only)
func updateSettings(w http.ResponseWriter, r *http.Request) {
	var settings Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate timezone
	if settings.ServerTimezone != "" {
		if _, err := time.LoadLocation(settings.ServerTimezone); err != nil {
			http.Error(w, "Invalid timezone: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		settings.ServerTimezone = "UTC"
	}

	_, err := db.Exec(`UPDATE settings SET
		alliance_name = ?,
		alliance_short_name = ?,
		award_first_points = ?,
		award_second_points = ?,
		award_third_points = ?,
		recommendation_points = ?,
		recent_conductor_penalty_days = ?,
		above_average_conductor_penalty = ?,
		r4r5_rank_boost = ?,
		first_time_conductor_boost = ?,
		schedule_message_template = ?,
		daily_message_template = ?,
		power_tracking_enabled = ?,
		server_timezone = ?,
		conductor_time = ?,
		backup_time = ?,
		display_timezones = ?,
		vs_points_daily_target = ?,
		vs_points_weekly_target = ?,
		min_power = ?,
		min_hq_level = ?,
		vip_seat_enabled = ?,
		marshal_guard_enabled = ?
		WHERE id = 1`,
		settings.AllianceName,
		settings.AllianceShortName,
		settings.AwardFirstPoints,
		settings.AwardSecondPoints,
		settings.AwardThirdPoints,
		settings.RecommendationPoints,
		settings.RecentConductorPenaltyDays,
		settings.AboveAverageConductorPenalty,
		settings.R4R5RankBoost,
		settings.FirstTimeConductorBoost,
		settings.ScheduleMessageTemplate,
		settings.DailyMessageTemplate,
		settings.PowerTrackingEnabled,
		settings.ServerTimezone,
		settings.ConductorTime,
		settings.BackupTime,
		settings.DisplayTimezones,
		settings.VSPointsDailyTarget,
		settings.VSPointsWeeklyTarget,
		settings.MinPower,
		settings.MinHQLevel,
		settings.VipSeatEnabled,
		settings.MarshalGuardEnabled,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Settings updated successfully"})
}

// Get backup rotation order (returns ordered member list with member details)
func getBackupRotation(w http.ResponseWriter, r *http.Request) {
	var rotationJSON string
	err := db.QueryRow(`SELECT COALESCE(backup_rotation_order, '[]') FROM settings WHERE id = 1`).Scan(&rotationJSON)
	if err != nil {
		rotationJSON = "[]"
	}

	var order []int
	json.Unmarshal([]byte(rotationJSON), &order)

	// Fetch all R4/R5 members
	rows, err := db.Query(`SELECT id, name, rank FROM members WHERE rank IN ('R4', 'R5') AND deleted_at IS NULL ORDER BY name`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type RotationMember struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Rank string `json:"rank"`
	}

	allR4R5 := []RotationMember{}
	for rows.Next() {
		var m RotationMember
		rows.Scan(&m.ID, &m.Name, &m.Rank)
		allR4R5 = append(allR4R5, m)
	}

	// Build ordered list: first those in rotation (in order), then others
	ordered := []RotationMember{}
	inRotation := map[int]bool{}
	memberByID := map[int]RotationMember{}
	for _, m := range allR4R5 {
		memberByID[m.ID] = m
	}
	for _, id := range order {
		if m, ok := memberByID[id]; ok {
			ordered = append(ordered, m)
			inRotation[id] = true
		}
	}
	for _, m := range allR4R5 {
		if !inRotation[m.ID] {
			ordered = append(ordered, m)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order":   order,
		"members": ordered,
	})
}

// Save backup rotation order
func saveBackupRotation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Order []int `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Expected JSON body with 'order' array", http.StatusBadRequest)
		return
	}
	if req.Order == nil {
		req.Order = []int{}
	}
	data, _ := json.Marshal(req.Order)
	_, err := db.Exec(`UPDATE settings SET backup_rotation_order = ? WHERE id = 1`, string(data))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Backup rotation saved"})
}

// Get train week mode (win/save)
func getTrainWeekMode(w http.ResponseWriter, r *http.Request) {
	var mode string
	err := db.QueryRow(`SELECT COALESCE(train_week_mode, 'win') FROM settings WHERE id = 1`).Scan(&mode)
	if err != nil {
		mode = "win"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"mode": mode})
}

// Set train week mode
func setTrainWeekMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Mode != "win" && req.Mode != "save") {
		http.Error(w, "Expected JSON body with 'mode': 'win' or 'save'", http.StatusBadRequest)
		return
	}
	_, err := db.Exec(`UPDATE settings SET train_week_mode = ? WHERE id = 1`, req.Mode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"mode": req.Mode})
}

// Lucky draw for Save Week mode
// POST /api/train-schedules/lucky-draw
// Body: { "type": "conductor"|"vip", "week": "YYYY-MM-DD" }
// Returns: { "winner": { id, name, rank }, "eligible": [...], "pool_size": N }
func luckyDraw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type string `json:"type"` // "conductor" or "vip"
		Week string `json:"week"` // ISO week start date YYYY-MM-DD
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Type != "conductor" && req.Type != "vip" {
		http.Error(w, "type must be 'conductor' or 'vip'", http.StatusBadRequest)
		return
	}
	if req.Week == "" {
		http.Error(w, "week is required (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	// Donation thresholds: conductor = 60k, vip = 40k
	threshold := int64(40000)
	if req.Type == "conductor" {
		threshold = int64(60000)
	}

	// Eligible = members with tech donations >= threshold for the given week
	// We look up the donations table. Since the donations table may not exist yet,
	// fall back to all eligible members if the table doesn't exist.
	type EligibleMember struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Rank string `json:"rank"`
	}

	eligible := []EligibleMember{}

	rows, err := db.Query(`
		SELECT m.id, m.name, m.rank
		FROM members m
		INNER JOIN tech_donations d ON d.member_id = m.id AND d.week_date = ?
		WHERE m.deleted_at IS NULL AND m.eligible = 1 AND d.amount >= ?
		ORDER BY m.name`, req.Week, threshold)

	if err != nil {
		// Donations table may not exist yet — fall back to all eligible members
		log.Printf("luckyDraw: donations query failed (%v), falling back to all eligible members", err)
		rows, err = db.Query(`SELECT id, name, rank FROM members WHERE deleted_at IS NULL AND eligible = 1 ORDER BY name`)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}
	defer rows.Close()

	for rows.Next() {
		var m EligibleMember
		rows.Scan(&m.ID, &m.Name, &m.Rank)
		eligible = append(eligible, m)
	}

	if len(eligible) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"winner":    nil,
			"eligible":  eligible,
			"pool_size": 0,
			"message":   "No eligible members found for this week",
		})
		return
	}

	// Pick a random winner
	idxBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(eligible))))
	winnerIdx := 0
	if err == nil {
		winnerIdx = int(idxBig.Int64())
	}
	winner := eligible[winnerIdx]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"winner":    winner,
		"eligible":  eligible,
		"pool_size": len(eligible),
	})
}

// isOfficerOrAdmin checks if the caller is R4, R5, or admin
func isOfficerOrAdmin(r *http.Request) bool {
	session, _ := store.Get(r, "session")
	if isAdmin, ok := session.Values["is_admin"].(bool); ok && isAdmin {
		return true
	}
	if memberID, ok := session.Values["member_id"].(int); ok {
		var rank string
		if err := db.QueryRow("SELECT rank FROM members WHERE id = ? AND deleted_at IS NULL", memberID).Scan(&rank); err == nil {
			return rank == "R4" || rank == "R5"
		}
	}
	return false
}

// GET /api/applicants — list all applicants
// rejection_reason is stripped unless caller is officer/admin
func getApplicants(w http.ResponseWriter, r *http.Request) {
	showRejectionReason := isOfficerOrAdmin(r)

	rows, err := db.Query(`
		SELECT a.id, a.name, a.power, a.rank, a.vouched_by, a.status,
		       u.username, a.applied_at, a.decided_at, a.trial_end_date,
		       a.notes, a.rejection_reason, a.member_id
		FROM applicants a
		LEFT JOIN users u ON u.id = a.decision_by_id
		ORDER BY CASE a.status
			WHEN 'pending'  THEN 1
			WHEN 'on_trial' THEN 2
			WHEN 'approved' THEN 3
			WHEN 'rejected' THEN 4
			ELSE 5 END, a.applied_at DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	applicants := make([]Applicant, 0)
	for rows.Next() {
		var a Applicant
		var power sql.NullInt64
		var rank, vouchedBy, decisionBy, decidedAt, trialEnd, notes, rejReason sql.NullString
		var memberID sql.NullInt64
		if err := rows.Scan(&a.ID, &a.Name, &power, &rank, &vouchedBy, &a.Status,
			&decisionBy, &a.AppliedAt, &decidedAt, &trialEnd,
			&notes, &rejReason, &memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if power.Valid {
			v := power.Int64
			a.Power = &v
		}
		if rank.Valid {
			a.Rank = &rank.String
		}
		if vouchedBy.Valid {
			a.VouchedBy = &vouchedBy.String
		}
		if decisionBy.Valid {
			a.DecisionBy = &decisionBy.String
		}
		if decidedAt.Valid {
			a.DecidedAt = &decidedAt.String
		}
		if trialEnd.Valid {
			a.TrialEndDate = &trialEnd.String
		}
		if notes.Valid {
			a.Notes = &notes.String
		}
		if rejReason.Valid && showRejectionReason {
			a.RejectionReason = &rejReason.String
		}
		if memberID.Valid {
			mid := int(memberID.Int64)
			a.MemberID = &mid
		}
		applicants = append(applicants, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(applicants)
}

// POST /api/applicants — create applicant (officers/admin)
func createApplicant(w http.ResponseWriter, r *http.Request) {
	var a Applicant
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	a.Name = strings.TrimSpace(a.Name)
	if a.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if a.Status == "" {
		a.Status = "pending"
	}

	result, err := db.Exec(`INSERT INTO applicants (name, power, rank, vouched_by, status, notes) VALUES (?, ?, ?, ?, ?, ?)`,
		a.Name,
		nullableInt64(a.Power),
		nullableString(a.Rank),
		nullableString(a.VouchedBy),
		a.Status,
		nullableString(a.Notes),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	a.ID = int(id)

	// Auto-create an R1 member record if created with on_trial or approved status
	if a.Status == "on_trial" || a.Status == "approved" {
		result2, err2 := db.Exec("INSERT INTO members (name, rank, eligible) VALUES (?, 'R1', 0)", a.Name)
		if err2 == nil {
			mid64, _ := result2.LastInsertId()
			mid := int(mid64)
			db.Exec("UPDATE applicants SET member_id = ? WHERE id = ?", mid, a.ID)
			a.MemberID = &mid
			log.Printf("Auto-created member id=%d (%s) for new applicant id=%d", mid, a.Name, a.ID)
		}
	}

	// Fetch applied_at
	db.QueryRow("SELECT applied_at FROM applicants WHERE id = ?", a.ID).Scan(&a.AppliedAt)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

// PUT /api/applicants/{id} — update applicant fields (officers/admin)
func updateApplicant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	var a Applicant
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	a.Name = strings.TrimSpace(a.Name)
	if a.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`UPDATE applicants SET name=?, power=?, rank=?, vouched_by=?, notes=? WHERE id=?`,
		a.Name,
		nullableInt64(a.Power),
		nullableString(a.Rank),
		nullableString(a.VouchedBy),
		nullableString(a.Notes),
		id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.ID = id

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

// PUT /api/applicants/{id}/status — set status, decision info (officers/admin)
func updateApplicantStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		Status          string  `json:"status"`
		TrialEndDate    *string `json:"trial_end_date,omitempty"`
		RejectionReason *string `json:"rejection_reason,omitempty"`
		MemberID        *int    `json:"member_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	valid := map[string]bool{"pending": true, "approved": true, "rejected": true, "on_trial": true}
	if !valid[req.Status] {
		http.Error(w, "status must be pending, approved, rejected, or on_trial", http.StatusBadRequest)
		return
	}

	// Determine decision_by_id from session
	session, _ := store.Get(r, "session")
	var decisionByID *int
	if req.Status != "pending" {
		if uid, ok := session.Values["user_id"].(int); ok {
			decisionByID = &uid
		}
	}

	var decidedAt *string
	if req.Status != "pending" {
		now := time.Now().UTC().Format(time.RFC3339)
		decidedAt = &now
	}

	_, err = db.Exec(`UPDATE applicants SET status=?, decision_by_id=?, decided_at=?, trial_end_date=?, rejection_reason=?, member_id=? WHERE id=?`,
		req.Status,
		nullableIntPtr(decisionByID),
		nullableStringPtr(decidedAt),
		nullableStringPtr(req.TrialEndDate),
		nullableStringPtr(req.RejectionReason),
		nullableIntPtr(req.MemberID),
		id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-create an R1 member record when applicant is put on trial or approved
	memberCreated := false
	var createdMemberID int
	if req.Status == "on_trial" || req.Status == "approved" {
		var existingMemberID *int
		var applicantName string
		_ = db.QueryRow("SELECT member_id, name FROM applicants WHERE id = ?", id).Scan(&existingMemberID, &applicantName)
		if existingMemberID == nil {
			result2, err2 := db.Exec("INSERT INTO members (name, rank, eligible) VALUES (?, 'R1', 0)", applicantName)
			if err2 == nil {
				mid64, _ := result2.LastInsertId()
				createdMemberID = int(mid64)
				db.Exec("UPDATE applicants SET member_id = ? WHERE id = ?", createdMemberID, id)
				memberCreated = true
				log.Printf("Auto-created member id=%d (%s) for applicant id=%d", createdMemberID, applicantName, id)
			}
		}
	}

	resp := map[string]interface{}{"status": req.Status}
	if memberCreated {
		resp["member_created"] = true
		resp["member_id"] = createdMemberID
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DELETE /api/applicants/{id} — delete applicant (R5/admin)
func deleteApplicant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	_, err = db.Exec("DELETE FROM applicants WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Applicant deleted"})
}

// Helper: nullable int64 from pointer
func nullableInt64(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// Helper: nullable string from pointer
func nullableString(v *string) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// Helper: nullable int from int pointer
func nullableIntPtr(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// Helper: nullable string from string pointer
func nullableStringPtr(v *string) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// Get member rankings with detailed score breakdown
func getMemberRankings(w http.ResponseWriter, r *http.Request) {
	// Always include all awards (active and inactive) - filtering is done on client side

	// Build ranking context using current date
	now := getServerTime()
	ctx, err := buildRankingContext(now)
	if err != nil {
		http.Error(w, "Failed to load ranking context: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get all active members
	rows, err := db.Query("SELECT id, name, nickname, rank FROM members WHERE deleted_at IS NULL ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		var nick sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &nick, &m.Rank); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if nick.Valid && nick.String != "" {
			m.Nickname = &nick.String
		}
		members = append(members, m)
	}

	// Load all award details with expired flag
	awardQuery := `
		SELECT 
			a.member_id, 
			a.award_type, 
			a.rank, 
			a.week_date,
			CASE 
				WHEN EXISTS (
					SELECT 1 FROM train_schedules ts
					WHERE (ts.conductor_id = a.member_id 
					       OR (ts.backup_id = a.member_id AND ts.conductor_showed_up = 0 AND ts.actual_conductor_id IS NULL)
					       OR ts.actual_conductor_id = a.member_id)
					AND ts.date >= a.week_date
				) THEN 1
				ELSE 0
			END as expired
		FROM awards a
		ORDER BY a.week_date DESC, a.rank ASC
	`

	awardRows, err := db.Query(awardQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer awardRows.Close()

	memberAwards := make(map[int][]AwardDetail)
	for awardRows.Next() {
		var memberID, rank, expired int
		var awardType, weekDate string
		if err := awardRows.Scan(&memberID, &awardType, &rank, &weekDate, &expired); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		points := 0
		switch rank {
		case 1:
			points = ctx.Settings.AwardFirstPoints
		case 2:
			points = ctx.Settings.AwardSecondPoints
		case 3:
			points = ctx.Settings.AwardThirdPoints
		}
		memberAwards[memberID] = append(memberAwards[memberID], AwardDetail{
			AwardType: awardType,
			Rank:      rank,
			Points:    points,
			WeekDate:  weekDate,
			Expired:   expired == 1,
		})
	}

	// Calculate rankings for each member
	rankings := make([]MemberRanking, 0)
	for _, member := range members {
		ranking := MemberRanking{
			Member:         member,
			AwardDetails:   memberAwards[member.ID],
			ConductorCount: 0,
		}

		// Calculate award points
		ranking.AwardPoints = ctx.AwardScoreMap[member.ID]

		// Calculate recommendation points
		recCount := ctx.RecommendationMap[member.ID]
		ranking.RecommendationCount = recCount
		ranking.RecommendationPoints = recCount * ctx.Settings.RecommendationPoints

		// Apply rank boost for R4/R5 members (exponential based on days since last conductor)
		if member.Rank == "R4" || member.Rank == "R5" {
			baseBoost := float64(ctx.Settings.R4R5RankBoost)

			// Calculate days since last conductor/backup duty
			var daysSinceLastDuty int = 0
			if stats, exists := ctx.ConductorStats[member.ID]; exists {
				var mostRecentDate *time.Time

				if stats.LastDate != nil {
					if lastDate, err := parseDate(*stats.LastDate); err == nil {
						mostRecentDate = &lastDate
					}
				}

				// Check if backup usage was more recent
				if stats.LastBackupUsed != nil {
					if backupDate, err := parseDate(*stats.LastBackupUsed); err == nil {
						if mostRecentDate == nil || backupDate.After(*mostRecentDate) {
							mostRecentDate = &backupDate
						}
					}
				}

				if mostRecentDate != nil {
					daysSinceLastDuty = int(now.Sub(*mostRecentDate).Hours() / 24)
				}
			}

			// Exponential formula: base_boost * 2^(days/7)
			multiplier := math.Pow(2, float64(daysSinceLastDuty)/7.0)
			ranking.RankBoost = int(math.Round(baseBoost * multiplier))
		}

		// Apply first time conductor boost if member has never been conductor
		if stats, exists := ctx.ConductorStats[member.ID]; !exists || stats.Count == 0 {
			// Calculate base score without first time boost
			baseScore := ranking.AwardPoints + ranking.RecommendationPoints + ranking.RankBoost
			if baseScore > 0 {
				ranking.FirstTimeConductorBoost = ctx.Settings.FirstTimeConductorBoost
			}
		}

		// Get conductor stats
		if stats, exists := ctx.ConductorStats[member.ID]; exists {
			ranking.ConductorCount = stats.Count
			ranking.LastConductorDate = stats.LastDate

			// Calculate above average penalty
			if float64(stats.Count) > ctx.AvgConductorCount {
				ranking.AboveAveragePenalty = ctx.Settings.AboveAverageConductorPenalty
			}

			// Calculate recent conductor penalty - check both conductor date and backup used date
			var mostRecentDate *time.Time

			if stats.LastDate != nil {
				if lastDate, err := parseDate(*stats.LastDate); err == nil {
					mostRecentDate = &lastDate
				}
			}

			// If they stepped in as backup, check if that's more recent
			if stats.LastBackupUsed != nil {
				if backupDate, err := parseDate(*stats.LastBackupUsed); err == nil {
					if mostRecentDate == nil || backupDate.After(*mostRecentDate) {
						mostRecentDate = &backupDate
					}
				}
			}

			// Apply penalty based on most recent duty (conductor or backup usage)
			if mostRecentDate != nil {
				daysSince := int(now.Sub(*mostRecentDate).Hours() / 24)
				ranking.DaysSinceLastConductor = &daysSince
				penalty := ctx.Settings.RecentConductorPenaltyDays - daysSince
				if penalty > 0 {
					ranking.RecentConductorPenalty = penalty
				}
			}
		}

		// Calculate total score using the same abstracted function
		ranking.TotalScore = calculateMemberScore(member, ctx)

		rankings = append(rankings, ranking)
	}

	// Load Marshal Guard stats per member
	mgRows, err := db.Query(`
		SELECT member_id, COUNT(DISTINCT event_id) as cnt, COALESCE(SUM(damage), 0) as total
		FROM marshal_guard_participants WHERE member_id IS NOT NULL GROUP BY member_id`)
	if err == nil {
		defer mgRows.Close()
		mgMap := map[int][2]int64{} // member_id -> [count, total_damage]
		for mgRows.Next() {
			var mid int
			var cnt int64
			var total int64
			if mgRows.Scan(&mid, &cnt, &total) == nil {
				mgMap[mid] = [2]int64{cnt, total}
			}
		}
		for i := range rankings {
			if stats, ok := mgMap[rankings[i].Member.ID]; ok {
				rankings[i].MGEventCount = int(stats[0])
				rankings[i].MGTotalDamage = stats[1]
			}
		}
	}

	// Sort by total score (highest first)
	for i := 0; i < len(rankings); i++ {
		for j := i + 1; j < len(rankings); j++ {
			if rankings[j].TotalScore > rankings[i].TotalScore {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rankings":                rankings,
		"settings":                ctx.Settings,
		"average_conductor_count": ctx.AvgConductorCount,
	})
}

// Get member point accumulation timelines over specified months
func getMemberTimelines(w http.ResponseWriter, r *http.Request) {
	// Parse months parameter (default to 3)
	monthsParam := r.URL.Query().Get("months")
	months := 3
	if monthsParam != "" {
		if m, err := strconv.Atoi(monthsParam); err == nil && m > 0 {
			months = m
		}
	}

	// Calculate start date (N months ago from today)
	now := getServerTime()
	startDate := now.AddDate(0, -months, 0)

	// Get all active members
	memberRows, err := db.Query("SELECT id, name, rank FROM members WHERE deleted_at IS NULL ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer memberRows.Close()

	var members []Member
	for memberRows.Next() {
		var m Member
		if err := memberRows.Scan(&m.ID, &m.Name, &m.Rank); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		members = append(members, m)
	}

	// Get settings for point calculations
	var settings Settings
	err = db.QueryRow(`SELECT award_first_points, award_second_points, award_third_points, 
		recommendation_points, r4r5_rank_boost, first_time_conductor_boost,
		recent_conductor_penalty_days, above_average_conductor_penalty 
		FROM settings WHERE id = 1`).Scan(
		&settings.AwardFirstPoints, &settings.AwardSecondPoints,
		&settings.AwardThirdPoints, &settings.RecommendationPoints,
		&settings.R4R5RankBoost, &settings.FirstTimeConductorBoost,
		&settings.RecentConductorPenaltyDays, &settings.AboveAverageConductorPenalty)
	if err != nil {
		http.Error(w, "Failed to load settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate average conductor count for above average penalty
	var totalConductors, memberCount int
	db.QueryRow(`
		SELECT COUNT(DISTINCT conductor_id), 
		       (SELECT COUNT(*) FROM members WHERE eligible = 1 AND deleted_at IS NULL)
		FROM train_schedules
	`).Scan(&totalConductors, &memberCount)
	avgConductorCount := 0.0
	if memberCount > 0 {
		avgConductorCount = float64(totalConductors) / float64(memberCount)
	}

	// Build timeline data for each member
	timelines := make(map[int]map[string]interface{})

	for _, member := range members {
		// Get train schedules where member was conductor (to track resets)
		conductorDates := []string{}
		conductorRows, err := db.Query(`
			SELECT date FROM train_schedules 
			WHERE (conductor_id = ? 
			       OR (backup_id = ? AND conductor_showed_up = 0 AND actual_conductor_id IS NULL)
			       OR actual_conductor_id = ?)
			AND date >= ?
			ORDER BY date ASC
		`, member.ID, member.ID, member.ID, formatDateString(startDate))
		if err != nil {
			continue
		}
		for conductorRows.Next() {
			var dateStr string
			if err := conductorRows.Scan(&dateStr); err == nil {
				conductorDates = append(conductorDates, dateStr)
			}
		}
		conductorRows.Close()

		// Get all awards and recommendations since start date (tracked separately)
		type PointEvent struct {
			Date   string
			Awards int
			Recs   int
		}
		eventMap := make(map[string]*PointEvent)

		// Get awards
		awardRows, err := db.Query(`
			SELECT week_date, rank FROM awards 
			WHERE member_id = ? AND week_date >= ?
			ORDER BY week_date ASC
		`, member.ID, formatDateString(startDate))
		if err == nil {
			for awardRows.Next() {
				var weekDate string
				var rank int
				if err := awardRows.Scan(&weekDate, &rank); err == nil {
					points := 0
					switch rank {
					case 1:
						points = settings.AwardFirstPoints
					case 2:
						points = settings.AwardSecondPoints
					case 3:
						points = settings.AwardThirdPoints
					}
					if eventMap[weekDate] == nil {
						eventMap[weekDate] = &PointEvent{Date: weekDate}
					}
					eventMap[weekDate].Awards += points
				}
			}
			awardRows.Close()
		}

		// Get recommendations (calculate non-linear points: 5*sqrt(n))
		recRows, err := db.Query(`
			SELECT DATE(created_at) as rec_date, COUNT(*) as count
			FROM recommendations 
			WHERE member_id = ? AND DATE(created_at) >= ?
			GROUP BY DATE(created_at)
			ORDER BY rec_date ASC
		`, member.ID, formatDateString(startDate))
		if err == nil {
			for recRows.Next() {
				var recDate string
				var count int
				if err := recRows.Scan(&recDate, &count); err == nil {
					// Non-linear formula: 5*sqrt(count)
					points := int(5 * math.Sqrt(float64(count)))
					if eventMap[recDate] == nil {
						eventMap[recDate] = &PointEvent{Date: recDate}
					}
					eventMap[recDate].Recs += points
				}
			}
			recRows.Close()
		}

		// Convert map to sorted slice
		var events []PointEvent
		for _, event := range eventMap {
			events = append(events, *event)
		}
		sort.Slice(events, func(i, j int) bool {
			return events[i].Date < events[j].Date
		})

		// Get power history for this member
		powerHistoryMap := make(map[string]int)
		powerRows, err := db.Query(`
			SELECT DATE(recorded_at) as power_date,
			       MAX(power) as max_power
			FROM power_history
			WHERE member_id = ? AND DATE(recorded_at) >= ?
			GROUP BY DATE(recorded_at)
			ORDER BY power_date ASC
		`, member.ID, formatDateString(startDate))
		if err == nil {
			for powerRows.Next() {
				var powerDate string
				var maxPower int
				if err := powerRows.Scan(&powerDate, &maxPower); err == nil {
					powerHistoryMap[powerDate] = maxPower
				}
			}
			powerRows.Close()
		}

		// Get VS points for this member (weekly totals keyed by Monday date)
		vsWeekMap := make(map[string]int)
		vsRows, err := db.Query(`
			SELECT week_date,
			       COALESCE(monday,0) + COALESCE(tuesday,0) + COALESCE(wednesday,0) +
			       COALESCE(thursday,0) + COALESCE(friday,0) + COALESCE(saturday,0) as weekly_total
			FROM vs_points
			WHERE member_id = ? AND week_date >= ?
			ORDER BY week_date ASC
		`, member.ID, formatDateString(startDate))
		if err == nil {
			for vsRows.Next() {
				var weekDate string
				var total int
				if err := vsRows.Scan(&weekDate, &total); err == nil {
					vsWeekMap[weekDate] = total
				}
			}
			vsRows.Close()
		}

		// Build weekly timeline arrays
		weekLabels := []string{}
		pointsWithReset := []int{}
		pointsCumulative := []int{}
		awardsWithReset := []int{}
		awardsCumulative := []int{}
		recsWithReset := []int{}
		recsCumulative := []int{}
		rankBoostWithReset := []int{}
		rankBoostCumulative := []int{}
		firstTimeBoostWithReset := []int{}
		firstTimeBoostCumulative := []int{}
		recentPenaltyWithReset := []int{}
		recentPenaltyCumulative := []int{}
		aboveAvgPenaltyWithReset := []int{}
		aboveAvgPenaltyCumulative := []int{}
		powerValues := []int{}
		lastKnownPower := 0
		vsWeeklyTotals := []int{}

		currentPoints := 0
		cumulativePoints := 0
		currentAwards := 0
		cumulativeAwards := 0
		currentRecs := 0
		cumulativeRecs := 0
		currentRankBoost := 0
		cumulativeRankBoost := 0
		currentFirstTimeBoost := 0
		cumulativeFirstTimeBoost := 0
		currentRecentPenalty := 0
		cumulativeRecentPenalty := 0
		currentAboveAvgPenalty := 0
		cumulativeAboveAvgPenalty := 0
		conductorIdx := 0
		conductorCountSoFar := 0

		// Generate week range from start to now (by Monday of each week)
		currentDate := getMondayOfWeek(startDate)
		for currentDate.Before(now) || currentDate.Equal(now) {
			weekStart := currentDate
			weekEnd := currentDate.AddDate(0, 0, 6)
			weekStartStr := formatDateString(weekStart)
			weekEndStr := formatDateString(weekEnd)

			// Format: "Jan 1 - Jan 7"
			weekLabel := fmt.Sprintf("%s - %s",
				weekStart.Format("Jan 2"),
				weekEnd.Format("Jan 2"))
			weekLabels = append(weekLabels, weekLabel)

			// Check if this week has a conductor event (train reset)
			weekHasReset := false
			for conductorIdx < len(conductorDates) && conductorDates[conductorIdx] <= weekEndStr {
				if conductorDates[conductorIdx] >= weekStartStr {
					weekHasReset = true
					conductorCountSoFar++
				}
				conductorIdx++
			}

			// Add points from events in this week
			weekAwards := 0
			weekRecs := 0
			for _, event := range events {
				if event.Date >= weekStartStr && event.Date <= weekEndStr {
					weekAwards += event.Awards
					weekRecs += event.Recs
				}
			}
			weekPoints := weekAwards + weekRecs

			currentPoints += weekPoints
			cumulativePoints += weekPoints
			currentAwards += weekAwards
			cumulativeAwards += weekAwards
			currentRecs += weekRecs
			cumulativeRecs += weekRecs

			// Calculate Rank Boost (R4/R5 exponential)
			weekRankBoost := 0
			if member.Rank == "R4" || member.Rank == "R5" {
				baseBoost := float64(settings.R4R5RankBoost)
				// Find most recent conductor date before this week
				daysSinceLastDuty := 0
				for i := conductorIdx - 1; i >= 0; i-- {
					if i < len(conductorDates) {
						if lastConductorDate, err := time.Parse("2006-01-02", conductorDates[i]); err == nil {
							daysSinceLastDuty = int(weekEnd.Sub(lastConductorDate).Hours() / 24)
							break
						}
					}
				}
				multiplier := math.Pow(2, float64(daysSinceLastDuty)/7.0)
				weekRankBoost = int(math.Round(baseBoost * multiplier))
			}
			currentRankBoost += weekRankBoost
			cumulativeRankBoost += weekRankBoost

			// Calculate First Time Conductor Boost
			weekFirstTimeBoost := 0
			if conductorCountSoFar == 0 {
				// Only if they have other points (awards, recs, or rank boost)
				if currentAwards > 0 || currentRecs > 0 || currentRankBoost > 0 {
					weekFirstTimeBoost = settings.FirstTimeConductorBoost
				}
			}
			currentFirstTimeBoost += weekFirstTimeBoost
			cumulativeFirstTimeBoost += weekFirstTimeBoost

			// Calculate Recent Conductor Penalty
			weekRecentPenalty := 0
			if conductorCountSoFar > 0 {
				// Find most recent conductor date
				for i := conductorIdx - 1; i >= 0; i-- {
					if i < len(conductorDates) {
						if lastConductorDate, err := time.Parse("2006-01-02", conductorDates[i]); err == nil {
							daysSince := int(weekEnd.Sub(lastConductorDate).Hours() / 24)
							penalty := settings.RecentConductorPenaltyDays - daysSince
							if penalty > 0 {
								weekRecentPenalty = penalty
							}
							break
						}
					}
				}
			}
			currentRecentPenalty += weekRecentPenalty
			cumulativeRecentPenalty += weekRecentPenalty

			// Calculate Above Average Penalty
			weekAboveAvgPenalty := 0
			if float64(conductorCountSoFar) > avgConductorCount {
				weekAboveAvgPenalty = settings.AboveAverageConductorPenalty
			}
			currentAboveAvgPenalty += weekAboveAvgPenalty
			cumulativeAboveAvgPenalty += weekAboveAvgPenalty

			// Apply reset at end of week if conductor event occurred
			if weekHasReset {
				currentPoints = 0
				currentAwards = 0
				currentRecs = 0
				currentRankBoost = 0
				currentFirstTimeBoost = 0
				currentRecentPenalty = 0
				currentAboveAvgPenalty = 0
			}

			// Find max power value for this week; fall back to last known value
			weekMaxPower := 0
			for powerDate, power := range powerHistoryMap {
				if powerDate >= weekStartStr && powerDate <= weekEndStr {
					if power > weekMaxPower {
						weekMaxPower = power
					}
				}
			}
			if weekMaxPower > 0 {
				lastKnownPower = weekMaxPower
			} else {
				weekMaxPower = lastKnownPower
			}

			pointsWithReset = append(pointsWithReset, currentPoints)
			pointsCumulative = append(pointsCumulative, cumulativePoints)
			awardsWithReset = append(awardsWithReset, currentAwards)
			awardsCumulative = append(awardsCumulative, cumulativeAwards)
			recsWithReset = append(recsWithReset, currentRecs)
			recsCumulative = append(recsCumulative, cumulativeRecs)
			rankBoostWithReset = append(rankBoostWithReset, currentRankBoost)
			rankBoostCumulative = append(rankBoostCumulative, cumulativeRankBoost)
			firstTimeBoostWithReset = append(firstTimeBoostWithReset, currentFirstTimeBoost)
			firstTimeBoostCumulative = append(firstTimeBoostCumulative, cumulativeFirstTimeBoost)
			recentPenaltyWithReset = append(recentPenaltyWithReset, currentRecentPenalty)
			recentPenaltyCumulative = append(recentPenaltyCumulative, cumulativeRecentPenalty)
			aboveAvgPenaltyWithReset = append(aboveAvgPenaltyWithReset, currentAboveAvgPenalty)
			aboveAvgPenaltyCumulative = append(aboveAvgPenaltyCumulative, cumulativeAboveAvgPenalty)
			powerValues = append(powerValues, weekMaxPower)
			vsWeeklyTotals = append(vsWeeklyTotals, vsWeekMap[weekStartStr])

			currentDate = currentDate.AddDate(0, 0, 7)
		}

		// Format conductor dates for display (convert to week labels)
		conductorWeekLabels := []string{}
		for _, condDate := range conductorDates {
			if parsedDate, err := time.Parse("2006-01-02", condDate); err == nil {
				monday := getMondayOfWeek(parsedDate)
				weekEnd := monday.AddDate(0, 0, 6)
				weekLabel := fmt.Sprintf("%s - %s",
					monday.Format("Jan 2"),
					weekEnd.Format("Jan 2"))
				conductorWeekLabels = append(conductorWeekLabels, weekLabel)
			}
		}

		timelines[member.ID] = map[string]interface{}{
			"dates":                        weekLabels,
			"points_with_reset":            pointsWithReset,
			"points_cumulative":            pointsCumulative,
			"awards_with_reset":            awardsWithReset,
			"awards_cumulative":            awardsCumulative,
			"recommendations_with_reset":   recsWithReset,
			"recommendations_cumulative":   recsCumulative,
			"rank_boost_with_reset":        rankBoostWithReset,
			"rank_boost_cumulative":        rankBoostCumulative,
			"first_time_boost_with_reset":  firstTimeBoostWithReset,
			"first_time_boost_cumulative":  firstTimeBoostCumulative,
			"recent_penalty_with_reset":    recentPenaltyWithReset,
			"recent_penalty_cumulative":    recentPenaltyCumulative,
			"above_avg_penalty_with_reset": aboveAvgPenaltyWithReset,
			"above_avg_penalty_cumulative": aboveAvgPenaltyCumulative,
			"conductor_dates":              conductorWeekLabels,
			"power":                        powerValues,
			"vs_weekly_total":              vsWeeklyTotals,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timelines)
}

// Generate weekly schedule message
func generateWeeklyMessage(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start")
	if startDate == "" {
		http.Error(w, "start date is required", http.StatusBadRequest)
		return
	}

	// Parse start date
	weekStart, err := parseDate(startDate)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	// Get settings
	settings, err := loadSettings()
	if err != nil {
		http.Error(w, "Failed to load settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get schedules for the week
	weekEnd := weekStart.AddDate(0, 0, 6)
	rows, err := db.Query(`
		SELECT 
			ts.date,
			COALESCE(m1.name, ts.conductor_name_snapshot, '[Removed]') as conductor_name,
			COALESCE(m2.name, ts.backup_name_snapshot, '[Removed]') as backup_name
		FROM train_schedules ts
		LEFT JOIN members m1 ON ts.conductor_id = m1.id AND m1.deleted_at IS NULL
		LEFT JOIN members m2 ON ts.backup_id = m2.id AND m2.deleted_at IS NULL
		WHERE ts.date >= ? AND ts.date <= ?
		ORDER BY ts.date
	`, formatDateString(weekStart), formatDateString(weekEnd))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var schedulesText strings.Builder
	for rows.Next() {
		var date, conductor, backup string
		if err := rows.Scan(&date, &conductor, &backup); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Parse date to get day name
		dateObj, _ := parseDate(date)
		dayName := dateObj.Format("Monday")

		schedulesText.WriteString(dayName + ": " + conductor + " (Backup: " + backup + ")\n")
	}

	// Build ranking context to get next 3 candidates
	ctx, err := buildRankingContext(weekStart)
	if err != nil {
		http.Error(w, "Failed to load ranking context: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get all eligible members and score them
	memberRows, err := db.Query("SELECT id, name, rank FROM members WHERE eligible = 1 AND deleted_at IS NULL ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer memberRows.Close()

	type ScoredMember struct {
		Name  string
		Score int
	}

	var scoredMembers []ScoredMember
	for memberRows.Next() {
		var m Member
		if err := memberRows.Scan(&m.ID, &m.Name, &m.Rank); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		score := calculateMemberScore(m, ctx)
		scoredMembers = append(scoredMembers, ScoredMember{
			Name:  m.Name,
			Score: score,
		})
	}

	// Sort by score (highest first)
	for i := 0; i < len(scoredMembers); i++ {
		for j := i + 1; j < len(scoredMembers); j++ {
			if scoredMembers[j].Score > scoredMembers[i].Score {
				scoredMembers[i], scoredMembers[j] = scoredMembers[j], scoredMembers[i]
			}
		}
	}

	// Get top 3
	var next3Text strings.Builder
	limit := 3
	if len(scoredMembers) < 3 {
		limit = len(scoredMembers)
	}
	for i := 0; i < limit; i++ {
		next3Text.WriteString(scoredMembers[i].Name + "\n")
	}

	// Format the message using template
	message := settings.ScheduleMessageTemplate
	message = strings.ReplaceAll(message, "{WEEK}", weekStart.Format("Jan 2, 2006"))
	message = strings.ReplaceAll(message, "{SCHEDULES}", schedulesText.String())
	message = strings.ReplaceAll(message, "{NEXT_3}", next3Text.String())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": message,
	})
}

// Generate daily message with conductor and backup for a specific date
func generateDailyMessage(w http.ResponseWriter, r *http.Request) {
	dateParam := r.URL.Query().Get("date")
	if dateParam == "" {
		http.Error(w, "date is required", http.StatusBadRequest)
		return
	}

	// Parse date
	date, err := parseDate(dateParam)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	// Get settings
	settings, err := loadSettings()
	if err != nil {
		http.Error(w, "Failed to load settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get schedule for the specific date
	var conductorName, conductorRank, backupName, backupRank string
	err = db.QueryRow(`
		SELECT 
			COALESCE(m1.name, ts.conductor_name_snapshot, '[Removed]'), COALESCE(m1.rank, ''),
			COALESCE(m2.name, ts.backup_name_snapshot, '[Removed]'), COALESCE(m2.rank, '')
		FROM train_schedules ts
		LEFT JOIN members m1 ON ts.conductor_id = m1.id AND m1.deleted_at IS NULL
		LEFT JOIN members m2 ON ts.backup_id = m2.id AND m2.deleted_at IS NULL
		WHERE ts.date = ?
	`, formatDateString(date)).Scan(&conductorName, &conductorRank, &backupName, &backupRank)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, "No schedule found for this date", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Format the message using template
	message := settings.DailyMessageTemplate
	message = strings.ReplaceAll(message, "{DATE}", date.Format("Monday, Jan 2, 2006"))
	message = strings.ReplaceAll(message, "{CONDUCTOR_NAME}", conductorName)
	message = strings.ReplaceAll(message, "{CONDUCTOR_RANK}", conductorRank)
	message = strings.ReplaceAll(message, "{BACKUP_NAME}", backupName)
	message = strings.ReplaceAll(message, "{BACKUP_RANK}", backupRank)

	// Replace dynamic time variables
	conductorTime := formatTimeAcrossTimezones(settings.ConductorTime, date, settings)
	backupTime := formatTimeAcrossTimezones(settings.BackupTime, date, settings)
	message = strings.ReplaceAll(message, "{CONDUCTOR_TIME}", conductorTime)
	message = strings.ReplaceAll(message, "{BACKUP_TIME}", backupTime)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": message,
	})
}

// Generate individual conductor reminder messages for the week
func generateConductorMessages(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start")
	if startDate == "" {
		http.Error(w, "start date is required", http.StatusBadRequest)
		return
	}

	// Load settings for timezone configuration
	settings, err := loadSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse start date
	weekStart, err := parseDate(startDate)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	// Get schedules for the week
	weekEnd := weekStart.AddDate(0, 0, 6)
	rows, err := db.Query(`
		SELECT 
			ts.date,
			COALESCE(m1.name, ts.conductor_name_snapshot, '[Removed]') as conductor_name
		FROM train_schedules ts
		LEFT JOIN members m1 ON ts.conductor_id = m1.id AND m1.deleted_at IS NULL
		WHERE ts.date >= ? AND ts.date <= ?
		ORDER BY ts.date
	`, formatDateString(weekStart), formatDateString(weekEnd))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Message variations for natural variety
	messageTemplates := []string{
		"Hi {NAME}! Just a reminder that you're the train conductor on {DAY}, {DATE}. Please be online around {CONDUCTOR_TIME} and ask in alliance chat for the train to be assigned to you. If anything comes up, let us know early so we can coordinate with the backup. Please add a reminder in your phone so you don't forget. Thanks for helping keep the train golden!",
		"Hi {NAME}! You're scheduled as train conductor on {DAY}, {DATE}. Please be online at {CONDUCTOR_TIME} and request the train in alliance chat. If your schedule changes, let us know in advance so we can coordinate with the backup. Add a reminder in your phone to make sure you're on time. Appreciate your support!",
		"Hi {NAME}! Just a heads-up that you're the train conductor on {DAY}, {DATE}. Please be online around {CONDUCTOR_TIME} and ask for the train in alliance chat. If you need help or need to swap, reach out early. Set a phone reminder so you don't miss it. Thanks a lot!",
		"Hi {NAME}! You're assigned as train conductor on {DAY}, {DATE}. Please be online at {CONDUCTOR_TIME} and request the train in alliance chat. If there are any timing issues, let us know so we can plan with the backup. Don't forget to add a reminder in your phone. Thanks for stepping up!",
		"Hi {NAME}! Reminder that you're the train conductor on {DAY}, {DATE}. Please be online around {CONDUCTOR_TIME} and ask in alliance chat for the train assignment. Let us know early if anything changes. Make sure to set a phone reminder. Much appreciated!",
		"Hi {NAME}! You're scheduled as train conductor on {DAY}, {DATE}. Please be online at {CONDUCTOR_TIME} and request the train in alliance chat. If you need assistance or a timing adjustment, just let us know. Add a phone reminder to help you remember. Thanks!",
		"Hi {NAME}! Just a reminder that you're the train conductor on {DAY}, {DATE}. Please be online around {CONDUCTOR_TIME} and ask in alliance chat for the train to be assigned. If anything comes up, please reach out early. Set a reminder in your phone so you're prepared. Thanks for helping the alliance!",
	}

	type DayMessage struct {
		Day     string `json:"day"`
		Name    string `json:"name"`
		Message string `json:"message"`
	}

	var messages []DayMessage
	templateIndex := 0

	for rows.Next() {
		var date, conductor string
		if err := rows.Scan(&date, &conductor); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Parse date to get day name and formatted date
		dateObj, _ := parseDate(date)
		dayName := dateObj.Format("Monday")
		dateFormatted := dateObj.Format("2 Jan") // e.g., "3 Jan" - unambiguous date format

		// Get template and cycle through them
		template := messageTemplates[templateIndex]
		templateIndex = (templateIndex + 1) % len(messageTemplates)

		// Replace placeholders
		message := strings.ReplaceAll(template, "{NAME}", conductor)
		message = strings.ReplaceAll(message, "{DAY}", dayName)
		message = strings.ReplaceAll(message, "{DATE}", dateFormatted)

		// Replace time placeholders with dynamic timezone-aware times
		conductorTime := formatTimeAcrossTimezones(settings.ConductorTime, dateObj, settings)
		message = strings.ReplaceAll(message, "{CONDUCTOR_TIME}", conductorTime)

		messages = append(messages, DayMessage{
			Day:     dayName,
			Name:    conductor,
			Message: message,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
	})
}

// R4/R5/Admin middleware - checks if user has R4, R5 rank or is admin
func r4r5Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session-name")
		memberID, ok := session.Values["member_id"].(int)
		if !ok {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		// Check if user is admin
		var isAdmin bool
		err := db.QueryRow("SELECT is_admin FROM users WHERE member_id = ?", memberID).Scan(&isAdmin)
		if err == nil && isAdmin {
			next(w, r)
			return
		}

		// Get member rank
		var rank string
		err = db.QueryRow("SELECT rank FROM members WHERE id = ? AND deleted_at IS NULL", memberID).Scan(&rank)
		if err != nil {
			http.Error(w, "Member not found", http.StatusNotFound)
			return
		}

		if rank != "R4" && rank != "R5" {
			http.Error(w, "Access denied - R4, R5 rank or admin privileges required", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// Get storm assignments
func getStormAssignments(w http.ResponseWriter, r *http.Request) {
	taskForce := r.URL.Query().Get("task_force")
	if taskForce == "" {
		taskForce = "A"
	}

	rows, err := db.Query(`
		SELECT id, task_force, building_id, member_id, position
		FROM storm_assignments
		WHERE task_force = ?
		ORDER BY building_id, position
	`, taskForce)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	assignments := []StormAssignment{}
	for rows.Next() {
		var a StormAssignment
		if err := rows.Scan(&a.ID, &a.TaskForce, &a.BuildingID, &a.MemberID, &a.Position); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		assignments = append(assignments, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assignments)
}

// Save storm assignments
func saveStormAssignments(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TaskForce   string `json:"task_force"`
		Assignments []struct {
			BuildingID string `json:"building_id"`
			MemberID   int    `json:"member_id"`
			Position   int    `json:"position"`
		} `json:"assignments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate task force
	if request.TaskForce != "A" && request.TaskForce != "B" {
		http.Error(w, "Invalid task force - must be A or B", http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete existing assignments for this task force
	_, err = tx.Exec("DELETE FROM storm_assignments WHERE task_force = ?", request.TaskForce)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Insert new assignments
	for _, assignment := range request.Assignments {
		_, err = tx.Exec(`
			INSERT INTO storm_assignments (task_force, building_id, member_id, position)
			VALUES (?, ?, ?, ?)
		`, request.TaskForce, assignment.BuildingID, assignment.MemberID, assignment.Position)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Assignments saved successfully",
	})
}

// Delete storm assignments for a task force
func deleteStormAssignments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskForce := vars["taskForce"]

	if taskForce != "A" && taskForce != "B" {
		http.Error(w, "Invalid task force - must be A or B", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM storm_assignments WHERE task_force = ?", taskForce)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Import member list from an alliance roster screenshot using OCR
func importMemberScreenshot(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed to read image", http.StatusBadRequest)
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read image data", http.StatusInternalServerError)
		return
	}

	// ── Attempt 1: row-based OCR using edge detection ─────────────────────────
	var ocrText string
	if decodedImg, _, decodeErr := image.Decode(bytes.NewReader(imageData)); decodeErr == nil {
		memberAttrs := analyzeScreenshot(decodedImg)
		if rowText, rowErr := extractMembersByRows(decodedImg, memberAttrs); rowErr == nil {
			rankCountRe := regexp.MustCompile(`(?i)\bR[1-5]\b`)
			if len(rankCountRe.FindAllString(rowText, -1)) >= 2 {
				ocrText = rowText
				log.Printf("Members OCR: using row-based text (%d rank tokens)", len(rankCountRe.FindAllString(rowText, -1)))
			}
		}
	}

	// ── Attempt 2: full-image OCR (original method) ────────────────────────────
	if ocrText == "" {
		// Preprocess and OCR
		processedData, err := preprocessImageForOCR(imageData)
		if err != nil {
			log.Printf("Image preprocessing failed: %v, using original", err)
			processedData = imageData
		}

		client := gosseract.NewClient()
		defer client.Close()

		if err := client.SetImageFromBytes(processedData); err != nil {
			http.Error(w, "Failed to load image for OCR", http.StatusInternalServerError)
			return
		}

		for _, mode := range []gosseract.PageSegMode{gosseract.PSM_AUTO, gosseract.PSM_SINGLE_BLOCK, gosseract.PSM_SPARSE_TEXT} {
			client.SetPageSegMode(mode)
			if t, err := client.Text(); err == nil && len(strings.TrimSpace(t)) > 0 {
				ocrText = t
				break
			}
		}
	}

	if strings.TrimSpace(ocrText) == "" {
		http.Error(w, "OCR failed: no text extracted from image", http.StatusUnprocessableEntity)
		return
	}
	log.Printf("Member list OCR text:\n%s\n---END OCR---", ocrText)

	// Parse OCR text for Name + Rank entries
	// Alliance roster lines typically look like: "R4  PlayerName  123,456,789"
	// or "PlayerName  R4" or "PlayerName (R4)"
	rankRe := regexp.MustCompile(`(?i)\bR([1-5])\b`)
	validRanks := map[string]bool{"R1": true, "R2": true, "R3": true, "R4": true, "R5": true}

	// Get existing active members
	existingMembers := make(map[string]Member)
	rows, dbErr := db.Query("SELECT id, name, rank FROM members WHERE deleted_at IS NULL")
	if dbErr == nil {
		defer rows.Close()
		for rows.Next() {
			var m Member
			rows.Scan(&m.ID, &m.Name, &m.Rank)
			existingMembers[strings.ToLower(m.Name)] = m
		}
	}

	detectedMembers := []DetectedMember{}
	parseErrors := []string{}
	seenNames := map[string]bool{}

	for _, line := range strings.Split(ocrText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := rankRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		rank := "R" + strings.ToUpper(matches[1])
		if !validRanks[rank] {
			continue
		}

		// Remove the rank token and surrounding noise (power numbers, special chars)
		cleaned := rankRe.ReplaceAllString(line, "")
		// Remove numbers that look like power (6+ digit sequences with optional commas)
		cleaned = regexp.MustCompile(`[\d,]{5,}`).ReplaceAllString(cleaned, "")
		// Remove leftover punctuation except letters/digits/spaces/hyphens/underscores/dots
		cleaned = regexp.MustCompile(`[^\p{L}\d\s\-_.]`).ReplaceAllString(cleaned, " ")
		name := strings.Join(strings.Fields(cleaned), " ")

		if len(name) < 2 {
			parseErrors = append(parseErrors, fmt.Sprintf("Could not extract name from line: %q", line))
			continue
		}

		nameLower := strings.ToLower(name)
		if seenNames[nameLower] {
			continue
		}
		seenNames[nameLower] = true

		detected := DetectedMember{Name: name, Rank: rank}

		if existing, found := existingMembers[nameLower]; found {
			if existing.Rank != rank {
				detected.RankChanged = true
				detected.OldRank = existing.Rank
			}
		} else {
			detected.IsNew = true
			var similar []string
			for existingLower, em := range existingMembers {
				if areSimilar(nameLower, existingLower) {
					similar = append(similar, em.Name)
				}
			}
			if len(similar) > 0 {
				detected.SimilarMatch = similar
			}
		}

		detectedMembers = append(detectedMembers, detected)
	}

	// Members in DB not detected in screenshot
	membersToRemove := []MemberToRemove{}
	for nameLower, em := range existingMembers {
		if !seenNames[nameLower] {
			membersToRemove = append(membersToRemove, MemberToRemove{ID: em.ID, Name: em.Name, Rank: em.Rank})
		}
	}

	result := map[string]interface{}{
		"detected_members":  detectedMembers,
		"members_to_remove": membersToRemove,
		"errors":            parseErrors,
		"total_rows":        len(detectedMembers),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// extractMembersByRows segments the member-list screenshot into individual rows
// and OCRs each row with PSM_SINGLE_LINE for cleaner rank-badge + name extraction.
//
// Returns a synthetic multi-line string where each line is the OCR output of one
// row.  The existing rank-regex parser in importMemberScreenshot consumes this
// directly, so the parsing logic is unchanged.
func extractMembersByRows(img image.Image, attrs *ScreenshotAttributes) (string, error) {
	bounds := img.Bounds()
	imgW := bounds.Dx()
	dataRegion := attrs.DataRegion

	grayFull := convertToGrayscale(img)

	const minRowH = 30
	rowBounds := findRowBoundaries(grayFull, dataRegion.Top, dataRegion.Bottom, minRowH)

	log.Printf("Members OCR: edge detection found %d rows in data region", len(rowBounds))

	var lines []string
	for i, rb := range rowBounds {
		rowTop, rowBottom := rb[0], rb[1]
		rowH := rowBottom - rowTop
		if rowH < minRowH {
			continue
		}

		// OCR the full row width — we want the rank badge text (R4, R3, …) which
		// appears as text at the left side of each row alongside the player name.
		rowImg := image.NewRGBA(image.Rect(0, 0, imgW, rowH))
		draw.Draw(rowImg, rowImg.Bounds(), img, image.Point{bounds.Min.X, rowTop}, draw.Src)

		scaled := scaleImage(rowImg, 3)
		gray := convertToGrayscale(scaled)

		var buf bytes.Buffer
		if err := png.Encode(&buf, gray); err != nil {
			log.Printf("Members row %d: encode failed: %v", i+1, err)
			continue
		}

		client := gosseract.NewClient()
		defer client.Close()
		client.SetImageFromBytes(buf.Bytes())
		client.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
		text, err := client.Text()
		if err != nil || len(strings.TrimSpace(text)) == 0 {
			continue
		}
		text = strings.TrimSpace(text)
		log.Printf("Members row %d: %q", i+1, text)
		lines = append(lines, text)
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("no member rows extracted")
	}

	return strings.Join(lines, "\n"), nil
}

// Auto-register a list of player names as R1 members (for unmatched OCR results)
func autoRegisterMembers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Names []string `json:"names"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Names) == 0 {
		http.Error(w, "Expected JSON body with 'names' array", http.StatusBadRequest)
		return
	}

	added := []string{}
	skipped := []string{}

	for _, name := range req.Names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Only insert if not already present
		var existing int
		err := db.QueryRow("SELECT COUNT(*) FROM members WHERE LOWER(name) = LOWER(?) AND deleted_at IS NULL", name).Scan(&existing)
		if err != nil || existing > 0 {
			skipped = append(skipped, name)
			continue
		}
		_, err = db.Exec("INSERT INTO members (name, rank, eligible) VALUES (?, 'R1', 1)", name)
		if err != nil {
			log.Printf("autoRegisterMembers: failed to insert %q: %v", name, err)
			skipped = append(skipped, name)
			continue
		}
		added = append(added, name)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"added":   added,
		"skipped": skipped,
		"message": fmt.Sprintf("Registered %d new member(s) as R1", len(added)),
	})
}

// Confirm and update members in database
func confirmMemberUpdates(w http.ResponseWriter, r *http.Request) {
	var request ConfirmRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := ConfirmResult{}

	// Process renames first
	for _, rename := range request.Renames {
		_, err := db.Exec("UPDATE members SET name = ? WHERE name = ?", rename.NewName, rename.OldName)
		if err != nil {
			log.Printf("Error renaming member %s to %s: %v", rename.OldName, rename.NewName, err)
			continue
		}
		log.Printf("Renamed member %s to %s", rename.OldName, rename.NewName)
	}

	// Create a set of member names from the request
	memberNames := make(map[string]bool)
	for _, member := range request.Members {
		memberNames[member.Name] = true
	}

	for _, member := range request.Members {
		// Check if member exists
		var existingID int
		var existingRank string
		err := db.QueryRow("SELECT id, rank FROM members WHERE name = ? AND deleted_at IS NULL", member.Name).Scan(&existingID, &existingRank)

		if err == sql.ErrNoRows {
			// Add new member
			_, err = db.Exec("INSERT INTO members (name, rank) VALUES (?, ?)", member.Name, member.Rank)
			if err != nil {
				log.Printf("Error adding member %s: %v", member.Name, err)
				continue
			}
			result.Added++
		} else if err == nil {
			// Update existing member if rank changed
			if existingRank != member.Rank {
				_, err = db.Exec("UPDATE members SET rank = ? WHERE id = ?", member.Rank, existingID)
				if err != nil {
					log.Printf("Error updating member %s: %v", member.Name, err)
					continue
				}
				result.Updated++
			} else {
				result.Unchanged++
			}
		}
	}

	// Remove specific members by ID if requested (soft delete)
	if len(request.RemoveMemberIDs) > 0 {
		today := formatDateString(getServerTime())
		for _, id := range request.RemoveMemberIDs {
			// Soft-delete the member
			_, err := db.Exec("UPDATE members SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL", id)
			if err != nil {
				log.Printf("Error removing member with id %d: %v", id, err)
				continue
			}
			// Expire open recommendations
			db.Exec("UPDATE recommendations SET expired = 1 WHERE member_id = ?", id)
			// Remove from future conductor schedules
			db.Exec("DELETE FROM train_schedules WHERE conductor_id = ? AND date >= ?", id, today)
			// Clear from future backup schedules
			db.Exec("UPDATE train_schedules SET backup_id = NULL, backup_name_snapshot = NULL WHERE backup_id = ? AND date >= ?", id, today)
			// Suspend linked user
			db.Exec("UPDATE users SET active = 0 WHERE member_id = ?", id)
			result.Removed++
		}
		log.Printf("Soft-deleted %d selected members", result.Removed)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Get power history for a specific member or all members
func getPowerHistory(w http.ResponseWriter, r *http.Request) {
	memberID := r.URL.Query().Get("member_id")
	limit := r.URL.Query().Get("limit")

	if limit == "" {
		limit = "30" // Default to last 30 records
	}

	var rows *sql.Rows
	var err error

	if memberID != "" {
		rows, err = db.Query(`
			SELECT ph.id, ph.member_id, ph.power, ph.recorded_at
			FROM power_history ph
			WHERE ph.member_id = ?
			ORDER BY ph.recorded_at DESC
			LIMIT ?
		`, memberID, limit)
	} else {
		rows, err = db.Query(`
			SELECT ph.id, ph.member_id, ph.power, ph.recorded_at
			FROM power_history ph
			ORDER BY ph.recorded_at DESC
			LIMIT ?
		`, limit)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	history := []PowerHistory{}
	for rows.Next() {
		var ph PowerHistory
		if err := rows.Scan(&ph.ID, &ph.MemberID, &ph.Power, &ph.RecordedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		history = append(history, ph)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// Add power record manually
func addPowerRecord(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MemberID int   `json:"member_id"`
		Power    int64 `json:"power"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if member exists and is active
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM members WHERE id = ? AND deleted_at IS NULL", request.MemberID).Scan(&exists)
	if err != nil || exists == 0 {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	result, err := db.Exec("INSERT INTO power_history (member_id, power) VALUES (?, ?)",
		request.MemberID, request.Power)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Power record added successfully",
		"id":      id,
	})
}

// ImageRegion represents a detected region in the screenshot
type ImageRegion struct {
	Name   string
	Top    int
	Bottom int
	Left   int
	Right  int
}

// ScreenshotAttributes contains analyzed attributes from the screenshot
type ScreenshotAttributes struct {
	Width          int
	Height         int
	TitleBarRegion *ImageRegion
	TabsRegion     *ImageRegion
	HeaderRegion   *ImageRegion
	DataRegion     *ImageRegion
	ButtonRegion   *ImageRegion
	RowHeight      int
	EstimatedRows  int
}

// Analyze screenshot to detect distinct regions and attributes
func analyzeScreenshot(img image.Image) *ScreenshotAttributes {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	attrs := &ScreenshotAttributes{
		Width:  width,
		Height: height,
	}

	// Detect dark title bar at top (typically 5-10% of height)
	// Title bars are usually dark colored
	titleBarHeight := height / 15 // ~6-7%
	if titleBarHeight < 30 {
		titleBarHeight = 30
	}
	attrs.TitleBarRegion = &ImageRegion{
		Name:   "TitleBar",
		Top:    0,
		Bottom: titleBarHeight,
		Left:   0,
		Right:  width,
	}

	// Detect tabs region (typically right below title bar, ~5-8% of height)
	tabsHeight := height / 15
	if tabsHeight < 40 {
		tabsHeight = 40
	}
	attrs.TabsRegion = &ImageRegion{
		Name:   "Tabs",
		Top:    titleBarHeight,
		Bottom: titleBarHeight + tabsHeight,
		Left:   0,
		Right:  width,
	}

	// Detect column headers (below tabs, ~5% of height)
	headerHeight := height / 20
	if headerHeight < 30 {
		headerHeight = 30
	}
	haederTop := titleBarHeight + tabsHeight
	attrs.HeaderRegion = &ImageRegion{
		Name:   "Headers",
		Top:    haederTop,
		Bottom: haederTop + headerHeight,
		Left:   0,
		Right:  width,
	}

	// Detect bottom button region (typically last 8-10% of height)
	buttonHeight := height / 10
	if buttonHeight < 50 {
		buttonHeight = 50
	}
	attrs.ButtonRegion = &ImageRegion{
		Name:   "BottomButton",
		Top:    height - buttonHeight,
		Bottom: height,
		Left:   0,
		Right:  width,
	}

	// Data region is everything between headers and bottom button
	dataTop := haederTop + headerHeight
	dataBottom := height - buttonHeight
	attrs.DataRegion = &ImageRegion{
		Name:   "DataRows",
		Top:    dataTop,
		Bottom: dataBottom,
		Left:   0,
		Right:  width,
	}

	// Estimate row height and count
	dataHeight := dataBottom - dataTop
	attrs.RowHeight = dataHeight / 10 // Assume ~10 visible rows
	if attrs.RowHeight < 40 {
		attrs.RowHeight = 40
	}
	attrs.EstimatedRows = dataHeight / attrs.RowHeight

	log.Printf("Screenshot Analysis: %dx%d, DataRegion: (%d,%d) to (%d,%d), Est. Rows: %d",
		width, height, attrs.DataRegion.Left, attrs.DataRegion.Top,
		attrs.DataRegion.Right, attrs.DataRegion.Bottom, attrs.EstimatedRows)

	return attrs
}

// Crop image to data region only
func cropToDataRegion(img image.Image, region *ImageRegion) image.Image {
	bounds := img.Bounds()
	top := region.Top
	bottom := region.Bottom
	left := region.Left
	right := region.Right

	// Ensure bounds are valid
	if top < bounds.Min.Y {
		top = bounds.Min.Y
	}
	if bottom > bounds.Max.Y {
		bottom = bounds.Max.Y
	}
	if left < bounds.Min.X {
		left = bounds.Min.X
	}
	if right > bounds.Max.X {
		right = bounds.Max.X
	}

	croppedImg := image.NewRGBA(image.Rect(0, 0, right-left, bottom-top))
	draw.Draw(croppedImg, croppedImg.Bounds(), img, image.Point{left, top}, draw.Src)

	log.Printf("Cropped image from %v to %v", bounds, croppedImg.Bounds())
	return croppedImg
}

// Convert image to grayscale
func convertToGrayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray.Set(x, y, img.At(x, y))
		}
	}

	return gray
}

// Enhance contrast using histogram equalization (simplified)
func enhanceContrast(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	enhanced := image.NewGray(bounds)

	// Calculate histogram
	histogram := make([]int, 256)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayVal := img.GrayAt(x, y).Y
			histogram[grayVal]++
		}
	}

	// Calculate cumulative distribution
	totalPixels := bounds.Dx() * bounds.Dy()
	cdf := make([]float64, 256)
	cdf[0] = float64(histogram[0]) / float64(totalPixels)
	for i := 1; i < 256; i++ {
		cdf[i] = cdf[i-1] + float64(histogram[i])/float64(totalPixels)
	}

	// Apply equalization
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayVal := img.GrayAt(x, y).Y
			newVal := uint8(cdf[grayVal] * 255)
			enhanced.SetGray(x, y, color.Gray{Y: newVal})
		}
	}

	return enhanced
}

// Apply adaptive thresholding to enhance text
func applyAdaptiveThreshold(img *image.Gray, blockSize int) *image.Gray {
	bounds := img.Bounds()
	thresholded := image.NewGray(bounds)

	halfBlock := blockSize / 2

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Calculate local mean in block
			sum := 0
			count := 0
			for by := y - halfBlock; by <= y+halfBlock; by++ {
				for bx := x - halfBlock; bx <= x+halfBlock; bx++ {
					if bx >= bounds.Min.X && bx < bounds.Max.X && by >= bounds.Min.Y && by < bounds.Max.Y {
						sum += int(img.GrayAt(bx, by).Y)
						count++
					}
				}
			}
			mean := uint8(sum / count)

			// Threshold: if pixel is darker than local mean, make it black, else white
			pixel := img.GrayAt(x, y).Y
			if pixel < mean-10 { // -10 for bias towards text
				thresholded.SetGray(x, y, color.Gray{Y: 0}) // Black (text)
			} else {
				thresholded.SetGray(x, y, color.Gray{Y: 255}) // White (background)
			}
		}
	}

	return thresholded
}

// Invert image (make text black on white background)
func invertImage(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	inverted := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayVal := img.GrayAt(x, y).Y
			inverted.SetGray(x, y, color.Gray{Y: 255 - grayVal})
		}
	}

	return inverted
}

// ── Edge-detection helpers ────────────────────────────────────────────────────
//
// These are used by all three OCR upload pipelines to:
//   1. Find exact row boundaries  (findRowBoundaries)
//   2. Locate the avatar/graphic column end  (detectAvatarEndX)
//
// Only Go stdlib is required — no OpenCV or external library.

// sobelMagnitude returns the Sobel gradient magnitude (0-255) at pixel (x,y)
// using a 3×3 kernel.  Pixels outside the image bounds are clamped to edge.
func sobelMagnitude(gray *image.Gray, x, y int) uint8 {
	b := gray.Bounds()
	clamp := func(v, lo, hi int) int {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}
	px := func(dx, dy int) int {
		return int(gray.GrayAt(clamp(x+dx, b.Min.X, b.Max.X-1), clamp(y+dy, b.Min.Y, b.Max.Y-1)).Y)
	}
	gx := -px(-1, -1) - 2*px(-1, 0) - px(-1, 1) + px(1, -1) + 2*px(1, 0) + px(1, 1)
	gy := -px(-1, -1) - 2*px(0, -1) - px(1, -1) + px(-1, 1) + 2*px(0, 1) + px(1, 1)
	mag := gx*gx + gy*gy
	// Fast integer sqrt via lookup; cap at 255
	if mag <= 0 {
		return 0
	}
	// approximate: mag = |gx|+|gy| (L1 norm, faster, good enough for thresholding)
	l1 := gx
	if l1 < 0 {
		l1 = -l1
	}
	abs_gy := gy
	if abs_gy < 0 {
		abs_gy = -abs_gy
	}
	l1 += abs_gy
	if l1 > 255*4 {
		return 255
	}
	return uint8(l1 / 4)
}

// regionEdgeDensity returns the mean Sobel magnitude (0.0–255.0) over the
// rectangle [x0,x1) × [y0,y1) of a grayscale image.
// Returns 0 if the rectangle is empty.
func regionEdgeDensity(gray *image.Gray, x0, y0, x1, y1 int) float64 {
	b := gray.Bounds()
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}
	if x1 <= x0 || y1 <= y0 {
		return 0
	}
	var sum int64
	for row := y0; row < y1; row++ {
		for col := x0; col < x1; col++ {
			sum += int64(sobelMagnitude(gray, col, row))
		}
	}
	n := int64((x1 - x0) * (y1 - y0))
	return float64(sum) / float64(n)
}

// findRowBoundaries scans the grayscale image between y=top and y=bottom,
// returning the y-coordinates (top edges) of each detected data row.
//
// Strategy: a separator line between rows is a horizontal band where almost
// every pixel has nearly the same brightness (variance < threshold).  We scan
// every row and mark "separator" rows, then collect the gaps between them as
// individual data rows.
//
// minRowH is the minimum acceptable row height in pixels; rows thinner than
// this are merged with their neighbour.
func findRowBoundaries(gray *image.Gray, top, bottom, minRowH int) [][2]int {
	b := gray.Bounds()
	if top < b.Min.Y {
		top = b.Min.Y
	}
	if bottom > b.Max.Y {
		bottom = b.Max.Y
	}
	if bottom-top < minRowH {
		return [][2]int{{top, bottom}}
	}

	width := b.Max.X - b.Min.X

	// For each scanline, compute the variance of pixel brightnesses.
	// A separator line has very low variance (solid colour).
	isSep := make([]bool, bottom-top)
	for y := top; y < bottom; y++ {
		var sum, sumSq int64
		for x := b.Min.X; x < b.Max.X; x++ {
			v := int64(gray.GrayAt(x, y).Y)
			sum += v
			sumSq += v * v
		}
		n := int64(width)
		mean := sum / n
		variance := sumSq/n - mean*mean
		// separator: very uniform (variance < 30) and not pure white (mean < 245)
		isSep[y-top] = variance < 30 && mean < 245
	}

	// Collect contiguous non-separator bands as rows
	var rows [][2]int
	inRow := false
	rowStart := 0
	for i, sep := range isSep {
		if !sep && !inRow {
			inRow = true
			rowStart = top + i
		} else if sep && inRow {
			h := top + i - rowStart
			if h >= minRowH {
				rows = append(rows, [2]int{rowStart, top + i})
			}
			inRow = false
		}
	}
	if inRow {
		h := bottom - rowStart
		if h >= minRowH {
			rows = append(rows, [2]int{rowStart, bottom})
		}
	}

	// Fallback: if no separators found, divide evenly
	if len(rows) == 0 {
		estimatedRows := (bottom - top) / max(minRowH, 1)
		if estimatedRows < 1 {
			estimatedRows = 1
		}
		rowH := (bottom - top) / estimatedRows
		for i := 0; i < estimatedRows; i++ {
			t := top + i*rowH
			rows = append(rows, [2]int{t, t + rowH})
		}
	}

	return rows
}

// detectAvatarEndX scans the row band [rowTop, rowBottom) left-to-right and
// returns the x coordinate where the avatar/graphic region ends.
//
// Avatars are complex artwork: high Sobel edge density.
// Text backgrounds are plain: low edge density.
//
// The scan uses a sliding vertical window of width windowW.  It returns the x
// where mean edge density in the window first drops below lowThresh after
// having been above highThresh (i.e. we've crossed from graphic → text).
//
// maxAvatarX is a hard cap (e.g. image width * 40 / 100) so the result is
// never unreasonably wide.  Returns maxAvatarX if no transition is found.
func detectAvatarEndX(gray *image.Gray, rowTop, rowBottom, maxAvatarX int) int {
	b := gray.Bounds()
	if rowTop < b.Min.Y {
		rowTop = b.Min.Y
	}
	if rowBottom > b.Max.Y {
		rowBottom = b.Max.Y
	}
	if rowBottom <= rowTop {
		return maxAvatarX
	}

	windowW := max((rowBottom-rowTop)/4, 4) // window width proportional to row height
	highThresh := 18.0                      // edge density considered "graphic"
	lowThresh := 8.0                        // edge density considered "text/plain"

	seenHigh := false
	for x := b.Min.X; x+windowW <= maxAvatarX; x += windowW / 2 {
		density := regionEdgeDensity(gray, x, rowTop, x+windowW, rowBottom)
		if density >= highThresh {
			seenHigh = true
		} else if seenHigh && density < lowThresh {
			// transition: graphic → plain — return the centre of the window
			result := x + windowW/2
			if result > maxAvatarX {
				return maxAvatarX
			}
			return result
		}
	}
	return maxAvatarX
}

// Scale image up for better OCR
func scaleImage(img image.Image, factor int) image.Image {
	bounds := img.Bounds()
	newWidth := bounds.Dx() * factor
	newHeight := bounds.Dy() * factor

	scaled := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Simple nearest-neighbor scaling
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			origX := x / factor
			origY := y / factor
			scaled.Set(x, y, img.At(origX, origY))
		}
	}

	return scaled
}

// Preprocess image for better OCR
func preprocessImageForOCR(imageData []byte) ([]byte, error) {
	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	log.Printf("Original image: %dx%d, format: %s", img.Bounds().Dx(), img.Bounds().Dy(), format)

	// Analyze screenshot to detect regions
	attrs := analyzeScreenshot(img)

	// Crop to data region only (remove UI elements)
	// For narrow screenshots or when power values might be cut off, use less aggressive cropping
	croppedImg := img
	if attrs.DataRegion != nil && attrs.Width > 600 {
		croppedImg = cropToDataRegion(img, attrs.DataRegion)
	} else {
		log.Printf("Skipping crop for narrow image to preserve power values")
	}

	// Scale up 2x for better OCR (small text is hard to read)
	scaledImg := scaleImage(croppedImg, 2)

	// Convert to grayscale
	grayImg := convertToGrayscale(scaledImg)

	// For small images, skip contrast enhancement (can make things worse)
	processedImg := grayImg

	// Encode back to bytes
	var buf bytes.Buffer
	if err := png.Encode(&buf, processedImg); err != nil {
		return nil, fmt.Errorf("failed to encode processed image: %v", err)
	}

	log.Printf("Image preprocessed: %dx%d -> %dx%d (2x scaled grayscale)",
		img.Bounds().Dx(), img.Bounds().Dy(),
		processedImg.Bounds().Dx(), processedImg.Bounds().Dy())

	return buf.Bytes(), nil
}

// Extract power data from image using OCR with preprocessing
func extractPowerDataFromImage(imageData []byte) ([]struct {
	MemberName string `json:"member_name"`
	Power      int64  `json:"power"`
}, error) {
	// ── Attempt 1: row-based extraction using edge detection ──────────────────
	img, _, decodeErr := image.Decode(bytes.NewReader(imageData))
	if decodeErr == nil {
		attrs := analyzeScreenshot(img)
		if rowRecords, rowErr := extractPowerByRows(img, attrs); rowErr == nil && len(rowRecords) >= 3 {
			log.Printf("Power OCR: row-based extraction succeeded with %d records", len(rowRecords))
			return rowRecords, nil
		} else {
			log.Printf("Power OCR: row-based extraction did not produce enough records (%d), falling back to full image", len(rowRecords))
		}
	}

	// ── Attempt 2: full-image OCR (original method) ────────────────────────────
	// Preprocess image to filter and enhance relevant regions
	processedData, err := preprocessImageForOCR(imageData)
	if err != nil {
		log.Printf("Warning: Image preprocessing failed: %v. Using original image.", err)
		processedData = imageData // Fallback to original
	}

	client := gosseract.NewClient()
	defer client.Close()

	err = client.SetImageFromBytes(processedData)
	if err != nil {
		return nil, fmt.Errorf("failed to load image: %v", err)
	}

	// Try different PSM modes for better recognition
	var text string
	psmModes := []gosseract.PageSegMode{
		gosseract.PSM_AUTO,
		gosseract.PSM_SINGLE_BLOCK,
		gosseract.PSM_SPARSE_TEXT,
	}

	for i, mode := range psmModes {
		client.SetPageSegMode(mode)
		extractedText, err := client.Text()
		if err == nil && len(strings.TrimSpace(extractedText)) > 0 {
			text = extractedText
			log.Printf("OCR successful with PSM mode %d (attempt %d)", mode, i+1)
			break
		}
		log.Printf("OCR attempt %d with PSM mode %d failed or empty", i+1, mode)
	}

	if len(strings.TrimSpace(text)) == 0 {
		return nil, fmt.Errorf("OCR failed: no text extracted after trying multiple modes")
	}

	// Log the extracted text for debugging
	log.Printf("OCR extracted text:\n%s\n---END OCR---", text)

	// Parse the OCR text
	records := parsePowerRankingsText(text)

	if len(records) == 0 {
		return nil, fmt.Errorf("no valid records found in extracted text (see server logs for OCR output)")
	}

	return records, nil
}

// Parse power rankings text (from OCR or manual input)
func parsePowerRankingsText(text string) []struct {
	MemberName string `json:"member_name"`
	Power      int64  `json:"power"`
} {
	var records []struct {
		MemberName string `json:"member_name"`
		Power      int64  `json:"power"`
	}

	lines := strings.Split(text, "\n")

	// Pattern specifically for Last War rankings format
	// Matches: optional rank badge (R4, R3), name (can have spaces), then large power number
	// Examples: "R4 Gary6126 77421000", "Nutty Tx 61926102", "R3 DYNOSUR 63785308"
	// Updated to better handle multi-word names
	rankPattern := regexp.MustCompile(`(?:R[0-9]\s+)?([A-Za-z][A-Za-z0-9_\s]+?)\s+([0-9]{7,})`)

	// Alternative simpler pattern: captures name with spaces followed by 7+ digit number
	simplePattern := regexp.MustCompile(`([A-Za-z][A-Za-z0-9_\s]+?)\s+([0-9]{7,})`)

	// Pattern for lines with rank number prefix: "1 Gary6126 R4 77421000" or "1 ileesu R4 66715876"
	rankPrefixPattern := regexp.MustCompile(`^[0-9]{1,3}\s+([A-Za-z][A-Za-z0-9_\s]+?)\s+(?:R[0-9]\s+)?([0-9]{7,})`)

	// Flexible pattern that allows letters in power (for OCR errors): "B 25) Nutty Tx s1926102"
	// This captures name followed by 7+ chars that may contain letters misread as digits
	flexiblePattern := regexp.MustCompile(`(?:[A-Z]{1,3}\s+)?(?:\d+\)?\s+)?([A-Za-z][A-Za-z0-9_\s]+?)\s+([A-Za-z0-9]{7,})`)

	// Track seen names to avoid duplicates from multi-line OCR
	seenNames := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip lines that are clearly UI elements or rank numbers
		if len(line) < 5 || regexp.MustCompile(`^[0-9]{1,2}$`).MatchString(line) {
			continue
		}

		// Skip common UI text
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "ranking") ||
			strings.Contains(lowerLine, "commander") ||
			strings.Contains(lowerLine, "power") ||
			strings.Contains(lowerLine, "kills") ||
			strings.Contains(lowerLine, "donation") {
			continue
		}

		// Try rank pattern first (for lines with R4, R3, etc.)
		matches := rankPattern.FindStringSubmatch(line)
		if len(matches) == 0 {
			// Try pattern with rank number prefix
			matches = rankPrefixPattern.FindStringSubmatch(line)
		}
		if len(matches) == 0 {
			// Try simple pattern
			matches = simplePattern.FindStringSubmatch(line)
		}
		if len(matches) == 0 {
			// Try flexible pattern (allows letters in power number for OCR errors)
			matches = flexiblePattern.FindStringSubmatch(line)
		}

		if len(matches) >= 3 {
			name := strings.TrimSpace(matches[1])
			// Clean up extra whitespace in names
			name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")

			powerStr := strings.ReplaceAll(matches[2], ",", "")
			powerStr = strings.ReplaceAll(powerStr, " ", "")
			powerStr = strings.ReplaceAll(powerStr, ".", "") // Remove periods OCR might insert
			// Common OCR character misreads for digits
			powerStr = strings.ReplaceAll(powerStr, "O", "0")
			powerStr = strings.ReplaceAll(powerStr, "o", "0")
			powerStr = strings.ReplaceAll(powerStr, "s", "6") // s often misread as 6
			powerStr = strings.ReplaceAll(powerStr, "S", "5") // S often misread as 5
			powerStr = strings.ReplaceAll(powerStr, "l", "1") // l often misread as 1
			powerStr = strings.ReplaceAll(powerStr, "I", "1") // I often misread as 1
			powerStr = strings.ReplaceAll(powerStr, "Z", "2") // Z sometimes misread as 2
			powerStr = strings.ReplaceAll(powerStr, "B", "8") // B sometimes misread as 8
			powerStr = strings.ReplaceAll(powerStr, "e", "6") // e sometimes misread as 6
			powerStr = strings.ReplaceAll(powerStr, "g", "9") // g sometimes misread as 9
			powerStr = strings.ReplaceAll(powerStr, "G", "6") // G sometimes misread as 6
			// Remove any remaining non-digit characters
			powerStr = regexp.MustCompile(`[^0-9]`).ReplaceAllString(powerStr, "")

			power, err := strconv.ParseInt(powerStr, 10, 64)

			// Validate: power should be realistic (1M to 1B range), name should be reasonable
			if err == nil && power >= 1000000 && power <= 9999999999 &&
				len(name) >= 3 && len(name) <= 30 && !seenNames[name] {
				records = append(records, struct {
					MemberName string `json:"member_name"`
					Power      int64  `json:"power"`
				}{
					MemberName: name,
					Power:      power,
				})
				seenNames[name] = true
				log.Printf("Parsed: %s -> %d", name, power)
			} else if err != nil {
				log.Printf("Failed to parse power for %s: %s (error: %v)", name, powerStr, err)
			}
		}
	}

	return records
}

// extractPowerByRows segments the power rankings screenshot into individual rows
// using edge-based separator detection and OCRs each name+power cell separately.
//
// Layout (approximate — confirmed by visual inspection of game screenshots):
//
//	|<-- 0-35% avatar+rank badge -->|<-- 35-80% player name -->|<-- 80-100% power -->|
//
// Falls back to the fixed 35% column if detectAvatarEndX returns a wider result.
func extractPowerByRows(img image.Image, attrs *ScreenshotAttributes) ([]struct {
	MemberName string `json:"member_name"`
	Power      int64  `json:"power"`
}, error) {
	bounds := img.Bounds()
	imgW := bounds.Dx()
	dataRegion := attrs.DataRegion

	grayFull := convertToGrayscale(img)

	const minRowH = 30
	rowBounds := findRowBoundaries(grayFull, dataRegion.Top, dataRegion.Bottom, minRowH)

	log.Printf("Power OCR: edge detection found %d rows in data region (was estimating %d)", len(rowBounds), attrs.EstimatedRows)

	records := []struct {
		MemberName string `json:"member_name"`
		Power      int64  `json:"power"`
	}{}

	nonDigitRe := regexp.MustCompile(`[^0-9]`)

	for i, rb := range rowBounds {
		rowTop, rowBottom := rb[0], rb[1]
		rowH := rowBottom - rowTop
		if rowH < minRowH {
			continue
		}

		// Detect avatar end; cap at 35% of image width
		nameStartX := detectAvatarEndX(grayFull, rowTop, rowBottom, imgW*35/100)
		nameEndX := imgW * 80 / 100
		powerStartX := imgW * 80 / 100
		if nameStartX >= nameEndX {
			nameStartX = imgW * 35 / 100 // fallback
		}

		// Extract name region (top 60% of row to avoid alliance tag if present)
		nameTopH := rowH * 60 / 100
		nameImg := image.NewRGBA(image.Rect(0, 0, nameEndX-nameStartX, nameTopH))
		draw.Draw(nameImg, nameImg.Bounds(), img, image.Point{nameStartX, rowTop}, draw.Src)

		// Extract power region (full row height, digits whitelist)
		powerImg := image.NewRGBA(image.Rect(0, 0, imgW-powerStartX, rowH))
		draw.Draw(powerImg, powerImg.Bounds(), img, image.Point{powerStartX, rowTop}, draw.Src)

		scaledName := scaleImage(nameImg, 3)
		grayName := convertToGrayscale(scaledName)
		scaledPower := scaleImage(powerImg, 3)
		grayPower := convertToGrayscale(scaledPower)

		var nameBuf bytes.Buffer
		if err := png.Encode(&nameBuf, grayName); err != nil {
			log.Printf("Power row %d: encode name failed: %v", i+1, err)
			continue
		}
		nameClient := gosseract.NewClient()
		defer nameClient.Close()
		nameClient.SetImageFromBytes(nameBuf.Bytes())
		nameClient.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
		nameText, err := nameClient.Text()
		if err != nil || len(strings.TrimSpace(nameText)) == 0 {
			continue
		}

		var powerBuf bytes.Buffer
		if err := png.Encode(&powerBuf, grayPower); err != nil {
			log.Printf("Power row %d: encode power failed: %v", i+1, err)
			continue
		}
		powerClient := gosseract.NewClient()
		defer powerClient.Close()
		powerClient.SetImageFromBytes(powerBuf.Bytes())
		powerClient.SetVariable("tessedit_char_whitelist", "0123456789,")
		powerClient.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
		powerText, err := powerClient.Text()
		if err != nil || len(strings.TrimSpace(powerText)) == 0 {
			log.Printf("Power row %d: Name='%s', no power value found", i+1, strings.TrimSpace(nameText))
			continue
		}

		name := cleanPlayerName(strings.TrimSpace(nameText))
		if len(name) < 2 {
			continue
		}

		powerStr := nonDigitRe.ReplaceAllString(strings.TrimSpace(powerText), "")
		power, err := strconv.ParseInt(powerStr, 10, 64)
		if err != nil || power < 1000000 { // power must be at least 1M
			log.Printf("Power row %d: skipping invalid power '%s' for '%s'", i+1, powerStr, name)
			continue
		}

		log.Printf("Power row %d: Name='%s', Power=%d", i+1, name, power)
		records = append(records, struct {
			MemberName string `json:"member_name"`
			Power      int64  `json:"power"`
		}{MemberName: name, Power: power})
	}

	return records, nil
}

// Normalize name for matching (remove common prefixes, spaces, special chars)
func normalizeName(name string) string {
	name = strings.ToLower(name)
	// Remove common prefixes
	name = strings.TrimPrefix(name, "the ")
	name = strings.TrimPrefix(name, "a ")
	name = strings.TrimPrefix(name, "an ")
	// Remove spaces and special characters
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "-", "")
	return name
}

// Calculate string similarity (0-100) using improved algorithm
func calculateSimilarity(s1, s2 string) int {
	// Normalize both strings
	n1 := normalizeName(s1)
	n2 := normalizeName(s2)

	// If normalized strings are identical, perfect match
	if n1 == n2 {
		return 100
	}

	// If one contains the other after normalization, very high score
	if strings.Contains(n1, n2) || strings.Contains(n2, n1) {
		return 90
	}

	// Calculate Levenshtein distance using existing function
	distance := levenshteinDistance(n1, n2)
	maxLen := len(n1)
	if len(n2) > maxLen {
		maxLen = len(n2)
	}

	if maxLen == 0 {
		return 0
	}

	// Convert distance to similarity percentage
	similarity := ((maxLen - distance) * 100) / maxLen

	return similarity
}

// Detect selected day tab by color (white/light background indicates selected tab)
func detectDayByColor(img image.Image) string {
	bounds := img.Bounds()
	width := bounds.Dx()

	// Days are arranged horizontally: Mon, Tues, Wed, Thur, Fri, Sat
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	tabWidth := width / 6 // Each tab takes ~1/6 of the width

	// Count white/light pixels in each tab region
	// Selected tab has white background, unselected tabs are gray
	lightCounts := make([]int, 6)

	for dayIdx := 0; dayIdx < 6; dayIdx++ {
		// Define the region for this day tab
		startX := dayIdx * tabWidth
		endX := startX + tabWidth
		if dayIdx == 5 {
			endX = width // Last tab goes to the end
		}

		// Sample the center 70% of each tab to avoid edge overlap issues
		// This prevents bleeding from adjacent tabs
		tabCenter := startX + tabWidth/2
		sampleWidth := int(float64(tabWidth) * 0.70)
		sampleStartX := tabCenter - sampleWidth/2
		sampleEndX := tabCenter + sampleWidth/2

		// Ensure we stay within bounds
		if sampleStartX < startX {
			sampleStartX = startX
		}
		if sampleEndX > endX {
			sampleEndX = endX
		}

		// Count white/light pixels in this region
		// Selected tab has white/cream background (high RGB values)
		lightCount := 0
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := sampleStartX; x < sampleEndX; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				// Convert from 16-bit to 8-bit color
				r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

				// Selected tab has a WHITE/cream background (all channels > 220).
				// Unselected tabs are medium gray (~145-165) which won't pass this threshold.
				if r8 > 220 && g8 > 220 && b8 > 220 {
					lightCount++
				}
			}
		}
		lightCounts[dayIdx] = lightCount
	}

	// Find the day with the most light pixels.
	// The selected tab should have significantly more white pixels than any unselected tab.
	maxLight := 0
	selectedDay := -1
	for i, count := range lightCounts {
		if count > maxLight {
			maxLight = count
			selectedDay = i
		}
	}

	// Require a minimum threshold to avoid false positives
	minThreshold := 50 // At least 50 bright pixels in the selected tab region
	if selectedDay >= 0 && maxLight > minThreshold {
		log.Printf("Day detected by color: %s (light pixel count: %d)", days[selectedDay], maxLight)
		log.Printf("Color counts per day: Mon=%d, Tue=%d, Wed=%d, Thu=%d, Fri=%d, Sat=%d",
			lightCounts[0], lightCounts[1], lightCounts[2], lightCounts[3], lightCounts[4], lightCounts[5])
		return days[selectedDay]
	}

	log.Printf("Color detection failed: max light count %d below threshold %d", maxLight, minThreshold)
	log.Printf("Color counts per day: Mon=%d, Tue=%d, Wed=%d, Thu=%d, Fri=%d, Sat=%d",
		lightCounts[0], lightCounts[1], lightCounts[2], lightCounts[3], lightCounts[4], lightCounts[5])
	return ""
}

// Extract just the day tab region and detect selected day by color
func detectDayFromTabRegion(imageData []byte) string {
	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		log.Printf("Failed to decode image for tab detection: %v", err)
		return ""
	}
	log.Printf("Image format for tab detection: %s, bounds: %v", format, img.Bounds())

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// The Mon-Sat day-tab row sits below the "Daily Rank / Weekly Rank" switcher.
	// Layout (approximate % of image height):
	//   0-6%   RANKING title bar
	//   7-12%  Daily Rank / Weekly Rank switcher tabs
	//  13-19%  Mon. Tues. Wed. Thur. Fri. Sat. day-selector tabs  ← target
	//  19-21%  Column-header row (Ranking | Commander | Points)
	//  21-92%  Data rows
	tabTop := int(float64(height) * 0.13)
	tabBottom := int(float64(height) * 0.19)

	// Ensure we don't go out of bounds
	if tabTop < 0 {
		tabTop = 0
	}
	if tabBottom > height {
		tabBottom = height
	}
	if tabBottom <= tabTop {
		tabBottom = tabTop + 100 // minimum 100px height
	}

	log.Printf("Extracting tab region: y=%d to y=%d (full image: %dx%d)", tabTop, tabBottom, width, height)

	// Create a new image with just the tab region
	tabRegion := image.NewRGBA(image.Rect(0, 0, width, tabBottom-tabTop))
	draw.Draw(tabRegion, tabRegion.Bounds(), img, image.Point{0, tabTop}, draw.Src)

	// First try: Detect by color (most reliable for this UI)
	dayByColor := detectDayByColor(tabRegion)
	if dayByColor != "" {
		return dayByColor
	}

	log.Printf("Color detection failed, falling back to OCR")

	// Fallback: Try OCR detection

	// Simple preprocessing for tab region: scale 2x and convert to grayscale
	// Don't use preprocessImageForOCR() as it tries to detect data regions (fails on small images)
	scaledTab := scaleImage(tabRegion, 2)
	grayTab := convertToGrayscale(scaledTab)

	// Convert to PNG bytes
	var buf bytes.Buffer
	if err := png.Encode(&buf, grayTab); err != nil {
		log.Printf("Failed to encode tab region: %v", err)
		return ""
	}

	log.Printf("Tab region preprocessed: %dx%d -> %dx%d (scaled 2x, grayscale)",
		tabRegion.Bounds().Dx(), tabRegion.Bounds().Dy(),
		grayTab.Bounds().Dx(), grayTab.Bounds().Dy())

	// Run OCR on the tab region
	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetImageFromBytes(buf.Bytes()); err != nil {
		log.Printf("Failed to load tab region for OCR: %v", err)
		return ""
	}

	client.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
	text, err := client.Text()
	if err != nil || len(strings.TrimSpace(text)) == 0 {
		log.Printf("Tab region OCR failed or empty")
		return ""
	}

	log.Printf("Tab region OCR text: %s", text)

	// Look for day names in the tab text
	textLower := strings.ToLower(text)
	days := []struct {
		name     string
		patterns []string
	}{
		{"monday", []string{"monday", "mon.", "mon"}},
		{"tuesday", []string{"tuesday", "tues.", "tues", "tue"}},
		{"wednesday", []string{"wednesday", "wed.", "wed"}},
		{"thursday", []string{"thursday", "thur.", "thur", "thu"}},
		{"friday", []string{"friday", "fri.", "fri"}},
		{"saturday", []string{"saturday", "sat.", "sat"}},
	}

	// Find which day patterns appear in the text
	// The selected tab usually appears first or more prominently
	for _, day := range days {
		for _, pattern := range day.patterns {
			if strings.Contains(textLower, pattern) {
				// Check if this appears early in the text (likely the selected/highlighted tab)
				idx := strings.Index(textLower, pattern)
				if idx < 100 { // Within first 100 chars suggests it's prominent
					log.Printf("Detected day '%s' from tab region (pattern: '%s' at position %d)", day.name, pattern, idx)
					return day.name
				}
			}
		}
	}

	return ""
}

// Extract VS points data from image and detect which day
func extractVSPointsDataFromImage(imageData []byte) (day string, records []struct {
	MemberName string `json:"member_name"`
	Points     int64  `json:"points"`
}, error error) {
	// First try to detect the day from the tab region specifically
	detectedDay := detectDayFromTabRegion(imageData)

	// If day detection failed, try text-based detection
	if detectedDay == "" {
		log.Printf("Tab region day detection failed, will try OCR fallback")
	}

	// Decode image for row segmentation
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode image: %v", err)
	}

	// Analyze screenshot to get regions
	attrs := analyzeScreenshot(img)

	// Try segmented OCR approach: extract and process individual rows
	log.Printf("Attempting row-by-row segmented OCR...")
	records, err = extractVSPointsByRows(img, attrs)

	// Validate row-based results: reject if any name contains a newline or a
	// known UI label (header row leaked in, alliance text, etc.)
	rowResultsValid := err == nil && len(records) >= 3
	if rowResultsValid {
		uiLabels := []string{"commander", "ranking", "points", "nova sapphire", "reset reapers"}
		for _, r := range records {
			name := strings.ToLower(r.MemberName)
			if strings.ContainsRune(r.MemberName, '\n') || strings.ContainsRune(r.MemberName, '\r') {
				log.Printf("Row OCR quality check: name %q contains newline — discarding row results", r.MemberName)
				rowResultsValid = false
				break
			}
			for _, label := range uiLabels {
				if strings.Contains(name, label) {
					log.Printf("Row OCR quality check: name %q matches UI label %q — discarding row results", r.MemberName, label)
					rowResultsValid = false
					break
				}
			}
			if !rowResultsValid {
				break
			}
		}
	}

	if !rowResultsValid {
		log.Printf("Segmented OCR failed or produced invalid results (%v), falling back to full image OCR", err)
		// Fallback to original full-image OCR approach
		records, err = extractVSPointsFullImage(imageData, attrs)
		if err != nil {
			return detectedDay, nil, err
		}
	}

	// If we didn't detect day from tab region, try text-based detection as fallback
	if detectedDay == "" {
		// Use current system day as last resort
		now := getServerTime()
		weekday := now.Weekday()
		switch weekday {
		case time.Monday:
			detectedDay = "monday"
		case time.Tuesday:
			detectedDay = "tuesday"
		case time.Wednesday:
			detectedDay = "wednesday"
		case time.Thursday:
			detectedDay = "thursday"
		case time.Friday:
			detectedDay = "friday"
		case time.Saturday:
			detectedDay = "saturday"
		default:
			// Sunday - default to Monday
			detectedDay = "monday"
		}
		log.Printf("Warning: Could not detect day from screenshot, using current system day: %s", detectedDay)
	}

	if len(records) == 0 {
		return "", nil, fmt.Errorf("no valid VS point records found in extracted text")
	}

	return detectedDay, records, nil
}

// Extract VS points by segmenting image into rows and OCR each row independently
func extractVSPointsByRows(img image.Image, attrs *ScreenshotAttributes) ([]struct {
	MemberName string `json:"member_name"`
	Points     int64  `json:"points"`
}, error) {
	bounds := img.Bounds()
	imgW := bounds.Dx()
	dataRegion := attrs.DataRegion

	// Convert full image to grayscale once — reused for edge analysis in every row.
	grayFull := convertToGrayscale(img)

	// Use separator-line scanning to find exact row boundaries instead of the
	// estimated rowHeight, which can misalign when rows vary in height.
	const minRowH = 30
	rowBounds := findRowBoundaries(grayFull, dataRegion.Top, dataRegion.Bottom, minRowH)

	log.Printf("VS OCR: edge detection found %d rows in data region (was estimating %d)", len(rowBounds), attrs.EstimatedRows)

	records := []struct {
		MemberName string `json:"member_name"`
		Points     int64  `json:"points"`
	}{}

	for i, rb := range rowBounds {
		rowTop, rowBottom := rb[0], rb[1]
		rowH := rowBottom - rowTop
		if rowH < minRowH {
			continue
		}

		// === Layout of each row in the VS ranking screenshot ===
		//
		//  |<-- avatar+rank -->|<---- name (top) / alliance (bottom) --->|<-- points -->|
		//
		// detectAvatarEndX measures the Sobel edge density across left-to-right column
		// slices: avatar art has high density, the text background is near-zero.
		// A hard cap at 40% prevents the detector from eating into the name column.
		nameStartX := detectAvatarEndX(grayFull, rowTop, rowBottom, imgW*40/100)

		nameEndX := imgW * 70 / 100
		pointsStartX := imgW * 70 / 100
		if nameStartX >= nameEndX {
			nameStartX = imgW * 28 / 100 // safety fallback to fixed column
		}

		nameTopH := rowH * 55 / 100 // top 55% of the row contains the player name

		// Extract name region: [nameStartX..nameEndX] × [rowTop..rowTop+nameTopH]
		nameImg := image.NewRGBA(image.Rect(0, 0, nameEndX-nameStartX, nameTopH))
		draw.Draw(nameImg, nameImg.Bounds(), img, image.Point{nameStartX, rowTop}, draw.Src)

		// Extract points region: [70%..100%] × full row height
		pointsImg := image.NewRGBA(image.Rect(0, 0, imgW-pointsStartX, rowH))
		draw.Draw(pointsImg, pointsImg.Bounds(), img, image.Point{pointsStartX, rowTop}, draw.Src)

		scaledName := scaleImage(nameImg, 3)
		grayName := convertToGrayscale(scaledName)

		scaledPoints := scaleImage(pointsImg, 3)
		grayPoints := convertToGrayscale(scaledPoints)

		// OCR name segment
		var nameBuf bytes.Buffer
		if err := png.Encode(&nameBuf, grayName); err != nil {
			log.Printf("Row %d: Failed to encode name segment: %v", i+1, err)
			continue
		}

		nameClient := gosseract.NewClient()
		defer nameClient.Close()
		nameClient.SetImageFromBytes(nameBuf.Bytes())
		nameClient.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
		nameText, err := nameClient.Text()
		if err != nil || len(strings.TrimSpace(nameText)) == 0 {
			continue // empty row
		}

		// OCR points segment — digits only
		var pointsBuf bytes.Buffer
		if err := png.Encode(&pointsBuf, grayPoints); err != nil {
			log.Printf("Row %d: Failed to encode points segment: %v", i+1, err)
			continue
		}

		pointsClient := gosseract.NewClient()
		defer pointsClient.Close()
		pointsClient.SetImageFromBytes(pointsBuf.Bytes())
		pointsClient.SetVariable("tessedit_char_whitelist", "0123456789,")
		pointsClient.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
		pointsText, err := pointsClient.Text()
		if err != nil || len(strings.TrimSpace(pointsText)) == 0 {
			log.Printf("Row %d: Name='%s', but no points found", i+1, strings.TrimSpace(nameText))
			continue
		}

		name := strings.TrimSpace(nameText)
		name = cleanPlayerName(name)

		nonDigitRe := regexp.MustCompile(`[^0-9]`)
		pointsStr := nonDigitRe.ReplaceAllString(strings.TrimSpace(pointsText), "")

		points, err := strconv.ParseInt(pointsStr, 10, 64)
		if err != nil {
			log.Printf("Row %d: Failed to parse points '%s' (raw: '%s'): %v", i+1, pointsStr, strings.TrimSpace(pointsText), err)
			continue
		}

		if points < 1000 { // skip header row / near-zero noise
			continue
		}

		log.Printf("Row %d: Name='%s', Points=%d", i+1, name, points)

		records = append(records, struct {
			MemberName string `json:"member_name"`
			Points     int64  `json:"points"`
		}{
			MemberName: name,
			Points:     points,
		})
	}

	return records, nil
}

// Fallback: Extract VS points from full image (original method)
func extractVSPointsFullImage(imageData []byte, attrs *ScreenshotAttributes) ([]struct {
	MemberName string `json:"member_name"`
	Points     int64  `json:"points"`
}, error) {
	// Preprocess image to filter and enhance relevant regions
	processedData, err := preprocessImageForOCR(imageData)
	if err != nil {
		log.Printf("Warning: Image preprocessing failed: %v. Using original image.", err)
		processedData = imageData // Fallback to original
	}

	client := gosseract.NewClient()
	defer client.Close()

	err = client.SetImageFromBytes(processedData)
	if err != nil {
		return nil, fmt.Errorf("failed to load image: %v", err)
	}

	// Try different PSM modes for better recognition
	var text string
	psmModes := []gosseract.PageSegMode{
		gosseract.PSM_AUTO,
		gosseract.PSM_SINGLE_BLOCK,
		gosseract.PSM_SPARSE_TEXT,
	}

	for i, mode := range psmModes {
		client.SetPageSegMode(mode)
		extractedText, err := client.Text()
		if err == nil && len(strings.TrimSpace(extractedText)) > 0 {
			text = extractedText
			log.Printf("OCR successful with PSM mode %d (attempt %d)", mode, i+1)
			break
		}
		log.Printf("OCR attempt %d with PSM mode %d failed or empty", i+1, mode)
	}

	if len(strings.TrimSpace(text)) == 0 {
		return nil, fmt.Errorf("OCR failed: no text extracted after trying multiple modes")
	}

	// Log the extracted text for debugging
	log.Printf("OCR extracted text:\n%s\n---END OCR---", text)

	// Parse the OCR text for VS points
	records := parseVSPointsText(text)

	return records, nil
}

// Clean player name by removing alliance tags, special characters, etc
func cleanPlayerName(name string) string {
	// Remove common OCR artifacts
	name = strings.ReplaceAll(name, "|", "I")
	name = strings.ReplaceAll(name, "~", "")
	name = strings.ReplaceAll(name, "`", "")

	// Remove alliance tags like [NTMs], (NTMs), etc.
	re := regexp.MustCompile(`\[.*?\]|\(.*?\)`)
	name = re.ReplaceAllString(name, "")

	// Remove rank numbers at the start (1), (2), etc.
	re = regexp.MustCompile(`^\d+\)?\s*`)
	name = re.ReplaceAllString(name, "")

	// Clean up whitespace
	name = strings.TrimSpace(name)

	return name
}

// Detect which day is selected from OCR text
// Looks for day names and context clues to determine the active tab
func detectSelectedDay(text string) string {
	textLower := strings.ToLower(text)

	// Check for presence of day names and "Daily Rank" which indicates VS points screen
	if !strings.Contains(textLower, "daily") && !strings.Contains(textLower, "rank") {
		return "" // Not a daily rank screen
	}

	// Count occurrences of each day name (the selected day often appears more prominently)
	days := map[string][]string{
		"monday":    {"monday", "mon.", "mon"},
		"tuesday":   {"tuesday", "tues.", "tues"},
		"wednesday": {"wednesday", "wed.", "wed"},
		"thursday":  {"thursday", "thur.", "thur"},
		"friday":    {"friday", "fri.", "fri"},
		"saturday":  {"saturday", "sat.", "sat"},
	}

	dayScores := make(map[string]int)

	for standardDay, variants := range days {
		for _, variant := range variants {
			// Count occurrences
			count := strings.Count(textLower, variant)
			dayScores[standardDay] += count

			// Higher weight if it appears near the beginning (likely the tab)
			if idx := strings.Index(textLower, variant); idx >= 0 && idx < 200 {
				dayScores[standardDay] += 2
			}
		}
	}

	// Find the day with the highest score
	maxScore := 0
	selectedDay := ""
	for day, score := range dayScores {
		if score > maxScore {
			maxScore = score
			selectedDay = day
		}
	}

	// Only return if we have strong confidence (score >= 3)
	if maxScore >= 3 {
		log.Printf("Detected day from full OCR text: %s (score: %d)", selectedDay, maxScore)
		return selectedDay
	}

	// Low confidence or no detection
	if maxScore > 0 {
		log.Printf("Low confidence day detection from OCR: %s (score: %d)", selectedDay, maxScore)
	}
	return ""
}

// Parse VS points text(from OCR or manual input)
func parseVSPointsText(text string) []struct {
	MemberName string `json:"member_name"`
	Points     int64  `json:"points"`
} {
	var records []struct {
		MemberName string `json:"member_name"`
		Points     int64  `json:"points"`
	}

	lines := strings.Split(text, "\n")

	// Number pattern: plain 6+ digits OR comma-formatted (19,291,992)
	numPat := `([0-9]{1,3}(?:,[0-9]{3})+|[0-9]{6,})`

	// Pattern 1: leading rank digit + name + optional alliance tag + number (all on one line)
	rankPrefixPattern := regexp.MustCompile(`^[0-9]{1,3}\s+([A-Za-z][A-Za-z0-9_\s]+?)\s+(?:\[[^\]]*\][^0-9]*)?\s*` + numPat)

	// Pattern 2: name then alliance tag then number (all on one line)
	alliancePattern := regexp.MustCompile(`([A-Za-z][A-Za-z0-9_\s]+?)\s+\[[^\]]*\][^0-9]*` + numPat)

	// Pattern 3: optional R-badge prefix, name, number (all on one line)
	rankPattern := regexp.MustCompile(`(?:R[0-9]\s+)?([A-Za-z][A-Za-z0-9_\s]*?)\s+` + numPat)

	// Pattern 4: simple name + number (all on one line)
	simplePattern := regexp.MustCompile(`([A-Za-z][A-Za-z0-9_\s]+?)\s+` + numPat)

	// Pattern to detect a line that contains only a number (possibly with OCR garbage before it)
	pointsOnlyPattern := regexp.MustCompile(`(?:^|[^A-Za-z])` + numPat + `\s*$`)

	// Helper to extract a plain number from a noisy line
	// e.g. "ﬂ T ) 160,689,845" → 160689845
	extractNumber := func(line string) (int64, bool) {
		m := pointsOnlyPattern.FindStringSubmatch(line)
		if len(m) < 2 {
			return 0, false
		}
		s := regexp.MustCompile(`[^0-9]`).ReplaceAllString(m[1], "")
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil || v < 10000 || v > 999999999 {
			return 0, false
		}
		return v, true
	}

	// Helper: does a line look like a player name (starts with a letter, length 3-25,
	// no big numbers, not a pure UI label)?
	isNameLike := func(line string) bool {
		if len(line) < 3 || len(line) > 35 {
			return false
		}
		if !regexp.MustCompile(`^[A-Za-z]`).MatchString(line) {
			return false
		}
		lower := strings.ToLower(line)
		for _, skip := range []string{"ranking", "commander", "points", "daily rank",
			"weekly rank", "your alliance", "nova sapphire", "reset reapers"} {
			if strings.Contains(lower, skip) {
				return false
			}
		}
		return true
	}

	// Pre-compiled helpers
	whitespaceRe := regexp.MustCompile(`\s+`)
	nonDigitRe := regexp.MustCompile(`[^0-9]`)
	skipLineRe := regexp.MustCompile(`^[0-9]{1,3}\.?$`)
	dayRe := regexp.MustCompile(`^(mon|tues|wed|thur|fri|sat|sun)\.?$`)

	// Track seen names to avoid duplicates
	seenNames := make(map[string]bool)

	addRecord := func(name string, points int64) {
		name = whitespaceRe.ReplaceAllString(strings.TrimSpace(name), " ")
		if len(name) >= 3 && len(name) <= 30 && !seenNames[name] &&
			points >= 10000 && points <= 999999999 {
			records = append(records, struct {
				MemberName string `json:"member_name"`
				Points     int64  `json:"points"`
			}{MemberName: name, Points: points})
			seenNames[name] = true
			log.Printf("Parsed VS points: %s -> %d", name, points)
		}
	}

	// ── Pass 1: same-line patterns (handles clean/manual input) ──────────────
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || len(line) < 5 || skipLineRe.MatchString(line) {
			continue
		}
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "ranking") || strings.Contains(lowerLine, "commander") ||
			strings.Contains(lowerLine, "points") || strings.Contains(lowerLine, "daily rank") ||
			strings.Contains(lowerLine, "weekly rank") || strings.Contains(lowerLine, "your alliance") ||
			dayRe.MatchString(lowerLine) {
			continue
		}

		var matches []string
		matches = rankPrefixPattern.FindStringSubmatch(line)
		if len(matches) == 0 {
			matches = alliancePattern.FindStringSubmatch(line)
		}
		if len(matches) == 0 {
			matches = rankPattern.FindStringSubmatch(line)
		}
		if len(matches) == 0 {
			matches = simplePattern.FindStringSubmatch(line)
		}

		if len(matches) >= 3 {
			name := strings.TrimSpace(matches[1])
			// Reject garbled OCR false positives: the matched name must contain
			// at least one word of length ≥ 3 (filters "B P ii", "ke i", etc.)
			hasLongWord := false
			for _, w := range whitespaceRe.Split(name, -1) {
				if len(w) >= 3 {
					hasLongWord = true
					break
				}
			}
			if !hasLongWord {
				continue
			}
			pointsStr := nonDigitRe.ReplaceAllString(strings.ReplaceAll(matches[2], ",", ""), "")
			points, err := strconv.ParseInt(pointsStr, 10, 64)
			if err == nil {
				addRecord(name, points)
			}
		}
	}

	// ── Pass 2: multi-line scan ────────────────────────────────────────────
	// For full-image OCR where name and points appear on adjacent lines.
	//
	// Walk each line right-to-left collecting consecutive "clean" tokens
	// (all-alphanumeric, length ≥ 3) as the candidate name.  Stop as soon
	// as a token is < 3 chars or contains non-alphanumeric characters.
	//
	//   "| =y B Bl Reddy sri"  → stops at "Bl"(2) → candidate = "Reddy sri"
	//   "L= > Al rahuld"       → stops at "Al"(2)  → candidate = "rahuld"
	//   "s i Patrick"          → stops at "i"(1)   → candidate = "Patrick"
	//   "&Y COL Geo222"        → stops at "&Y"(!)  → candidate = "COL Geo222"
	//   "CAIOVLF"              → full line          → candidate = "CAIOVLF"
	if len(records) < 3 {
		cleanWordRe := regexp.MustCompile(`^[A-Za-z0-9]+$`)

		extractSuffix := func(line string) string {
			parts := whitespaceRe.Split(strings.TrimSpace(line), -1)
			var nameParts []string
			for i := len(parts) - 1; i >= 0; i-- {
				w := parts[i]
				if len(w) < 3 || !cleanWordRe.MatchString(w) {
					break
				}
				nameParts = append([]string{w}, nameParts...)
			}
			return strings.Join(nameParts, " ")
		}

		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			candidateName := extractSuffix(line)
			if !isNameLike(candidateName) || seenNames[candidateName] {
				continue
			}
			for j := i + 1; j <= i+3 && j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" {
					continue
				}
				pts, ok := extractNumber(nextLine)
				if ok {
					cleanName := cleanPlayerName(candidateName)
					if cleanName != "" {
						addRecord(cleanName, pts)
					}
					break
				}
			}
		}
	}

	return records
}

// HTTP handler to process VS points screenshot
func processVSPointsScreenshot(w http.ResponseWriter, r *http.Request) {
	var records []struct {
		MemberName string `json:"member_name"`
		Points     int64  `json:"points"`
	}
	var detectedDay string
	var weekDate string

	// Check if this is a multipart form (image upload) or JSON (manual text)
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Handle image upload
		err := r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// Get the week parameter (optional, defaults to "current")
		weekParam := r.FormValue("week")
		if weekParam == "" {
			weekParam = "current"
		}

		file, _, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "No image file provided", http.StatusBadRequest)
			return
		}
		defer file.Close()

		imageData, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read image", http.StatusInternalServerError)
			return
		}

		detectedDay, records, err = extractVSPointsDataFromImage(imageData)
		if err != nil {
			http.Error(w, fmt.Sprintf("OCR processing failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Determine the week date based on the week parameter
		now := getServerTime()
		if weekParam == "last" {
			// Subtract 7 days to get last week
			now = now.AddDate(0, 0, -7)
		}
		weekday := now.Weekday()
		daysFromMonday := int(weekday) - 1
		if weekday == time.Sunday {
			daysFromMonday = 6
		}
		monday := now.AddDate(0, 0, -daysFromMonday)
		weekDate = monday.Format("2006-01-02")
	} else {
		// Handle JSON (manual text or pre-parsed data)
		var request struct {
			Records []struct {
				MemberName string `json:"member_name"`
				Points     int64  `json:"points"`
			} `json:"records"`
			Text string `json:"text"` // Raw text to parse
			Day  string `json:"day"`  // Optional: specify the day
			Week string `json:"week"` // Optional: "current" or "last"
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if request.Text != "" {
			// Parse raw text
			records = parseVSPointsText(request.Text)
			detectedDay = detectSelectedDay(request.Text)
		} else {
			records = request.Records
		}

		// Use provided day if available, otherwise use detected day
		if request.Day != "" {
			detectedDay = strings.ToLower(request.Day)
		}

		// Determine the week date based on the week parameter
		weekParam := request.Week
		if weekParam == "" {
			weekParam = "current"
		}
		now := getServerTime()
		if weekParam == "last" {
			// Subtract 7 days to get last week
			now = now.AddDate(0, 0, -7)
		}
		weekday := now.Weekday()
		daysFromMonday := int(weekday) - 1
		if weekday == time.Sunday {
			daysFromMonday = 6
		}
		monday := now.AddDate(0, 0, -daysFromMonday)
		weekDate = monday.Format("2006-01-02")
	}

	if len(records) == 0 {
		http.Error(w, "No valid VS point records found", http.StatusBadRequest)
		return
	}

	if detectedDay == "" {
		http.Error(w, "Could not determine which day these VS points are for. Please specify the day manually.", http.StatusBadRequest)
		return
	}

	// Normalize day name
	dayColumn := detectedDay
	validDays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	isValidDay := false
	for _, d := range validDays {
		if dayColumn == d {
			isValidDay = true
			break
		}
	}
	if !isValidDay {
		http.Error(w, fmt.Sprintf("Invalid day: %s. Must be monday-saturday", dayColumn), http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	successCount := 0
	notFoundMembers := []string{}
	updatedMembers := []string{}

	for _, record := range records {
		// Try to find member by exact name match first
		var memberID int
		var memberName string
		err := tx.QueryRow("SELECT id, name FROM members WHERE LOWER(name) = LOWER(?) AND deleted_at IS NULL", record.MemberName).Scan(&memberID, &memberName)

		if err == sql.ErrNoRows {
			// Try fuzzy matching (also checks nicknames)
			rows, err := tx.Query("SELECT id, name, nickname FROM members WHERE deleted_at IS NULL")
			if err != nil {
				continue
			}

			bestMatch := ""
			bestScore := 0
			bestID := 0

			for rows.Next() {
				var id int
				var name string
				var nickname sql.NullString
				if err := rows.Scan(&id, &name, &nickname); err != nil {
					continue
				}

				score := calculateSimilarity(record.MemberName, name)
				// Also score against nickname if present
				if nickname.Valid && nickname.String != "" {
					if ns := calculateSimilarity(record.MemberName, nickname.String); ns > score {
						score = ns
					}
				}
				if score > bestScore {
					bestScore = score
					bestMatch = name
					bestID = id
				}
			}
			rows.Close()

			// Use fuzzy match if similarity is high enough (70%+)
			if bestScore >= 70 {
				memberID = bestID
				memberName = bestMatch
				log.Printf("Fuzzy matched '%s' to '%s' (similarity: %d%%)", record.MemberName, bestMatch, bestScore)
			} else {
				notFoundMembers = append(notFoundMembers, record.MemberName)
				continue
			}
		}

		// Upsert VS points for this member
		var existingID int
		err = tx.QueryRow("SELECT id FROM vs_points WHERE member_id = ? AND week_date = ?",
			memberID, weekDate).Scan(&existingID)

		if err == sql.ErrNoRows {
			// Insert new record with the specific day's points
			query := fmt.Sprintf(`
				INSERT INTO vs_points (member_id, week_date, %s, updated_at)
				VALUES (?, ?, ?, CURRENT_TIMESTAMP)`, dayColumn)
			_, err = tx.Exec(query, memberID, weekDate, record.Points)
		} else if err == nil {
			// Update existing record with the specific day's points
			query := fmt.Sprintf(`
				UPDATE vs_points 
				SET %s = ?, updated_at = CURRENT_TIMESTAMP
				WHERE member_id = ? AND week_date = ?`, dayColumn)
			_, err = tx.Exec(query, record.Points, memberID, weekDate)
		}

		if err != nil {
			log.Printf("Failed to save VS points for %s: %v", memberName, err)
			continue
		}

		successCount++
		updatedMembers = append(updatedMembers, memberName)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to save changes", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":         fmt.Sprintf("Successfully updated VS points for %d members on %s", successCount, detectedDay),
		"day":             detectedDay,
		"week_date":       weekDate,
		"success_count":   successCount,
		"updated_members": updatedMembers,
	}

	if len(notFoundMembers) > 0 {
		response["not_found_members"] = notFoundMembers
		response["warning"] = fmt.Sprintf("%d members could not be matched to the database", len(notFoundMembers))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Process screenshot data with OCR support
func processPowerScreenshot(w http.ResponseWriter, r *http.Request) {
	// Check if power tracking is enabled
	var powerTrackingEnabled bool
	err := db.QueryRow("SELECT COALESCE(power_tracking_enabled, 0) FROM settings WHERE id = 1").Scan(&powerTrackingEnabled)
	if err != nil || !powerTrackingEnabled {
		http.Error(w, "Power tracking is not enabled", http.StatusForbidden)
		return
	}

	var records []struct {
		MemberName string `json:"member_name"`
		Power      int64  `json:"power"`
	}

	// Check if this is a multipart form (image upload) or JSON (manual text)
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Handle image upload
		err := r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "No image file provided", http.StatusBadRequest)
			return
		}
		defer file.Close()

		imageData, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read image", http.StatusInternalServerError)
			return
		}

		records, err = extractPowerDataFromImage(imageData)
		if err != nil {
			http.Error(w, fmt.Sprintf("OCR processing failed: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		// Handle JSON (manual text or pre-parsed data)
		var request struct {
			Records []struct {
				MemberName string `json:"member_name"`
				Power      int64  `json:"power"`
			} `json:"records"`
			Text string `json:"text"` // Raw text to parse
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if request.Text != "" {
			// Parse raw text
			records = parsePowerRankingsText(request.Text)
		} else {
			records = request.Records
		}
	}

	if len(records) == 0 {
		http.Error(w, "No valid records found", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	successCount := 0
	failedCount := 0
	errors := []string{}

	// Get all member names (and nicknames) for fuzzy matching
	allMembers := []struct {
		ID       int
		Name     string
		Nickname string
	}{}
	rows, err := tx.Query("SELECT id, name, COALESCE(nickname, '') FROM members WHERE deleted_at IS NULL")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var m struct {
				ID       int
				Name     string
				Nickname string
			}
			if rows.Scan(&m.ID, &m.Name, &m.Nickname) == nil {
				allMembers = append(allMembers, m)
			}
		}
	}

	for _, record := range records {
		// Try exact match first
		var memberID int
		err := tx.QueryRow("SELECT id FROM members WHERE name = ? AND deleted_at IS NULL", record.MemberName).Scan(&memberID)

		if err != nil {
			// Try case-insensitive match
			err = tx.QueryRow("SELECT id FROM members WHERE LOWER(name) = LOWER(?) AND deleted_at IS NULL", record.MemberName).Scan(&memberID)
		}

		if err != nil {
			// Try fuzzy matching with Levenshtein-like similarity (also checks nicknames)
			bestMatch := ""
			bestMatchID := 0
			bestScore := 0

			for _, member := range allMembers {
				score := calculateSimilarity(record.MemberName, member.Name)
				// Also score against nickname if present
				if member.Nickname != "" {
					if ns := calculateSimilarity(record.MemberName, member.Nickname); ns > score {
						score = ns
					}
				}
				log.Printf("Comparing '%s' with '%s' (nickname: '%s'): score=%d%%", record.MemberName, member.Name, member.Nickname, score)
				if score > bestScore {
					bestScore = score
					bestMatch = member.Name
					bestMatchID = member.ID
				}
			}

			if bestMatchID > 0 && bestScore >= 50 { // Lowered to 50% similarity threshold
				memberID = bestMatchID
				log.Printf("✓ Fuzzy matched '%s' to '%s' (score: %d%%)", record.MemberName, bestMatch, bestScore)
			} else {
				failedCount++
				if bestMatch != "" {
					errors = append(errors, fmt.Sprintf("Member '%s' not found (closest: '%s' at %d%%, need 50%%+)", record.MemberName, bestMatch, bestScore))
				} else {
					errors = append(errors, fmt.Sprintf("Member '%s' not found (no members in database)", record.MemberName))
				}
				continue
			}
		}

		// Insert power record
		_, err = tx.Exec("INSERT INTO power_history (member_id, power) VALUES (?, ?)",
			memberID, record.Power)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Failed to add record for '%s': %v", record.MemberName, err))
			continue
		}

		successCount++
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to save records", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       fmt.Sprintf("Processed %d records successfully, %d failed", successCount, failedCount),
		"success_count": successCount,
		"failed_count":  failedCount,
		"errors":        errors,
	})
}

// ============================================================
// Marshal Guard handlers
// ============================================================

type MarshalGuardEvent struct {
	ID                  int                       `json:"id"`
	EventDate           string                    `json:"event_date"`
	TotalAllianceDamage int64                     `json:"total_alliance_damage"`
	Notes               string                    `json:"notes"`
	CreatedAt           string                    `json:"created_at"`
	CreatedByID         *int                      `json:"created_by_id"`
	ParticipantCount    int                       `json:"participant_count,omitempty"`
	TopDamageDealer     string                    `json:"top_damage_dealer,omitempty"`
	TopDamage           int64                     `json:"top_damage,omitempty"`
	Participants        []MarshalGuardParticipant `json:"participants,omitempty"`
}

type MarshalGuardParticipant struct {
	ID           int    `json:"id"`
	EventID      int    `json:"event_id"`
	MemberID     *int   `json:"member_id"`
	MemberName   string `json:"member_name,omitempty"`
	NameSnapshot string `json:"name_snapshot"`
	AllianceTag  string `json:"alliance_tag"`
	RankInEvent  int    `json:"rank_in_event"`
	Damage       int64  `json:"damage"`
	AttackCount  *int   `json:"attack_count"`
}

type MarshalGuardOCRResult struct {
	EventDate       string             `json:"event_date"`
	TotalDamage     int64              `json:"total_damage"`
	Participants    []MGOCRParticipant `json:"participants"`
	ExistingEventID *int               `json:"existing_event_id,omitempty"`
}

type MGOCRParticipant struct {
	RankInEvent  int    `json:"rank_in_event"`
	NameSnapshot string `json:"name_snapshot"`
	AllianceTag  string `json:"alliance_tag"`
	Damage       int64  `json:"damage"`
	AttackCount  *int   `json:"attack_count"`
	MemberID     *int   `json:"member_id"`
	MemberName   string `json:"member_name,omitempty"`
}

type MGConfirmRequest struct {
	EventDate        string             `json:"event_date"`
	TotalDamage      int64              `json:"total_damage"`
	Notes            string             `json:"notes"`
	OverwriteEventID *int               `json:"overwrite_event_id,omitempty"`
	Participants     []MGOCRParticipant `json:"participants"`
}

type MGMemberStats struct {
	MemberID    int     `json:"member_id"`
	MemberName  string  `json:"member_name"`
	MemberRank  string  `json:"member_rank"`
	EventCount  int     `json:"event_count"`
	TotalDamage int64   `json:"total_damage"`
	AvgRank     float64 `json:"avg_rank"`
	BestDamage  int64   `json:"best_damage"`
}

// V2 OCR preview types (mg_segment pipeline)

type MGV2PreviewRow struct {
	Rank           int     `json:"rank"`
	Name           string  `json:"name"`         // player name without alliance tag
	AllianceTag    string  `json:"alliance_tag"` // e.g. "RSRP"
	NameOK         bool    `json:"name_ok"`
	DamageStr      string  `json:"damage_str"` // e.g. "27.35G"
	Damage         int64   `json:"damage"`
	DamageOK       bool    `json:"damage_ok"`
	RankFixed      bool    `json:"rank_fixed"`
	MemberID       *int    `json:"member_id,omitempty"`
	MemberName     string  `json:"member_name,omitempty"`
	GraveyardMatch bool    `json:"graveyard_match,omitempty"`
	SourceFileIdx  *int    `json:"source_file_idx,omitempty"` // index into uploaded files
	CropY0Pct      float64 `json:"crop_y0_pct,omitempty"`     // row crop top (0.0–1.0)
	CropY1Pct      float64 `json:"crop_y1_pct,omitempty"`     // row crop bottom (0.0–1.0)
}

type MGV2PreviewEvent struct {
	EventDate         string           `json:"event_date"`
	TopPlayerName     string           `json:"top_player_name"`
	TopPlayerDmgStr   string           `json:"top_player_damage_str"`
	TopPlayerDmg      int64            `json:"top_player_damage"`
	Rows              []MGV2PreviewRow `json:"rows"`
	ExistingEventID   *int             `json:"existing_event_id,omitempty"`
	SourceFileIndices []int            `json:"source_file_indices,omitempty"`
}

// POST /api/marshal-guard/process-mg-v2 — OCR using the mg_segment pipeline.
// Accepts multiple screenshots, groups by event date, returns one preview
// object per distinct date so the UI can review/edit before importing.
func processMGV2(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(30 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["images[]"]
	if len(files) == 0 {
		files = r.MultipartForm.File["image"]
	}
	if len(files) == 0 {
		http.Error(w, "No images provided", http.StatusBadRequest)
		return
	}
	if len(files) > 20 {
		http.Error(w, "Maximum 20 images allowed", http.StatusBadRequest)
		return
	}

	// Process each image and collect results keyed by event date.
	type eventAccum struct {
		topName     string
		topDmgStr   string
		topDmgInt   int64
		members     map[int]*mgMemberOCR // rank → best OCR result
		fileIndices []int                // which input file indices contributed
	}
	byDate := map[string]*eventAccum{}
	dateOrder := []string{}

	for fileIdx, fh := range files {
		f, err := fh.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		imgResult, err := mgProcessImage(data)
		if err != nil {
			log.Printf("processMGV2: %v", err)
			continue
		}

		date := imgResult.EventDate
		if date == "" {
			date = "unknown"
		}
		acc, exists := byDate[date]
		if !exists {
			acc = &eventAccum{members: map[int]*mgMemberOCR{}}
			byDate[date] = acc
			dateOrder = append(dateOrder, date)
		}
		acc.fileIndices = append(acc.fileIndices, fileIdx)
		if acc.topName == "" && imgResult.TopPlayerName != "" {
			acc.topName = imgResult.TopPlayerName
		}
		if imgResult.TopPlayerDmgInt > acc.topDmgInt {
			acc.topDmgStr = imgResult.TopPlayerDmgStr
			acc.topDmgInt = imgResult.TopPlayerDmgInt
		}
		for i := range imgResult.Members {
			m := &imgResult.Members[i]
			existing, has := acc.members[m.Rank]
			if !has || m.DamageInt > existing.DamageInt || (!existing.NameOK && m.NameOK) {
				cp := *m
				cp.FileIdx = fileIdx // record which uploaded file this row came from
				acc.members[m.Rank] = &cp
			}
		}
	}

	// Load members for matching.
	allMembers, membErr := loadAllMembers()
	log.Printf("processMGV2: loaded %d members (err=%v)", len(allMembers), membErr)
	deletedMembers, _ := loadDeletedMembers()
	var knownTag string
	db.QueryRow(`SELECT COALESCE(alliance_short_name, '') FROM settings WHERE id = 1`).Scan(&knownTag)
	knownTag = strings.ToUpper(strings.TrimSpace(knownTag))

	// Build preview events.
	var events []MGV2PreviewEvent
	for _, date := range dateOrder {
		acc := byDate[date]

		// Sort ranks.
		ranks := make([]int, 0, len(acc.members))
		for rank := range acc.members {
			ranks = append(ranks, rank)
		}
		sort.Ints(ranks)

		var rows []MGV2PreviewRow
		// Fill gaps between min and max rank.
		if len(ranks) > 0 {
			minRank := ranks[0]
			maxRank := ranks[len(ranks)-1]
			memberIdx := 0
			for rank := minRank; rank <= maxRank; rank++ {
				if memberIdx < len(ranks) && ranks[memberIdx] == rank {
					m := acc.members[rank]
					// Parse alliance tag and plain name from "[TAG]PlayerName".
					allianceTag, nameOnly := parsePlayerTag(m.Name, knownTag)
					srcIdx := m.FileIdx
					row := MGV2PreviewRow{
						Rank:          rank,
						Name:          nameOnly,
						AllianceTag:   allianceTag,
						NameOK:        m.NameOK,
						DamageStr:     m.DamageStr,
						Damage:        m.DamageInt,
						DamageOK:      m.DamageOK,
						RankFixed:     m.RankFixed,
						SourceFileIdx: &srcIdx,
						CropY0Pct:     m.CropY0,
						CropY1Pct:     m.CropY1,
					}
					// Match to member using plain name (without alliance tag).
					if allMembers != nil {
						fake := MGOCRParticipant{NameSnapshot: nameOnly}
						matchMGParticipant(&fake, allMembers)
						row.MemberID = fake.MemberID
						row.MemberName = fake.MemberName
					}
					// If still unmatched, check the graveyard (deleted members).
					if row.MemberID == nil && deletedMembers != nil {
						fake := MGOCRParticipant{NameSnapshot: nameOnly}
						matchMGParticipant(&fake, deletedMembers)
						if fake.MemberID != nil {
							row.MemberID = fake.MemberID
							row.MemberName = fake.MemberName
							row.GraveyardMatch = true
						}
					}
					rows = append(rows, row)
					memberIdx++
				} else {
					// Gap row — rank not detected in any screenshot.
					rows = append(rows, MGV2PreviewRow{Rank: rank})
				}
			}
		}

		// Add the top player as rank 1.
		if acc.topName != "" {
			topTag, topName := parsePlayerTag(acc.topName, knownTag)
			topRow := MGV2PreviewRow{
				Rank:        1,
				Name:        topName,
				AllianceTag: topTag,
				NameOK:      topName != "",
				DamageStr:   acc.topDmgStr,
				Damage:      acc.topDmgInt,
				DamageOK:    acc.topDmgInt > 0,
			}
			if topName != "" {
				fake := MGOCRParticipant{NameSnapshot: topName}
				if allMembers != nil {
					matchMGParticipant(&fake, allMembers)
					topRow.MemberID = fake.MemberID
					topRow.MemberName = fake.MemberName
				}
				if topRow.MemberID == nil && deletedMembers != nil {
					fake2 := MGOCRParticipant{NameSnapshot: topName}
					matchMGParticipant(&fake2, deletedMembers)
					if fake2.MemberID != nil {
						topRow.MemberID = fake2.MemberID
						topRow.MemberName = fake2.MemberName
						topRow.GraveyardMatch = true
					}
				}
			}
			rows = append([]MGV2PreviewRow{topRow}, rows...)
		}

		// Fix rows where a lower-ranked player has higher damage than the one above —
		// almost always a missing decimal point in the OCR output (e.g. 237G → 2.37G).
		// Rank 1 is skipped in the top-down pass because its OCR is less reliable;
		// a wrongly-low rank 1 value would cascade bad corrections into rank 2+ rows.
		prevDamage := int64(-1)
		for i := range rows {
			r := &rows[i]
			if r.Rank == 1 || r.Damage == 0 {
				continue // skip rank 1 and gap/unread rows
			}
			if prevDamage >= 0 && r.Damage > prevDamage {
				fixed := r.Damage / 100
				if fixed > 0 && fixed <= prevDamage {
					r.Damage = fixed
					r.DamageStr = mgFormatDamageStr(fixed)
				}
			}
			if r.Damage > 0 {
				prevDamage = r.Damage
			}
		}
		// Validate rank 1: top player's damage must be ≥ rank 2. If not, flag for review.
		if len(rows) >= 2 && rows[0].Rank == 1 && rows[1].Damage > 0 && rows[0].Damage > 0 && rows[0].Damage < rows[1].Damage {
			rows[0].DamageOK = false
		}

		// Check for existing event.
		var existingID *int
		if date != "unknown" {
			var eid int
			if err := db.QueryRow(`SELECT id FROM marshal_guard_events WHERE event_date = ?`, date).Scan(&eid); err == nil {
				existingID = &eid
			}
		}

		events = append(events, MGV2PreviewEvent{
			EventDate:         date,
			TopPlayerName:     acc.topName,
			TopPlayerDmgStr:   acc.topDmgStr,
			TopPlayerDmg:      acc.topDmgInt,
			Rows:              rows,
			ExistingEventID:   existingID,
			SourceFileIndices: acc.fileIndices,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// GET /api/marshal-guard — list events with summary
func listMarshalGuardEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT e.id, e.event_date, e.total_alliance_damage, COALESCE(e.notes, ''),
			e.created_at, e.created_by_id,
			COUNT(p.id) as participant_count,
			COALESCE((SELECT p2.name_snapshot FROM marshal_guard_participants p2 WHERE p2.event_id = e.id ORDER BY p2.damage DESC LIMIT 1), '') as top_dealer,
			COALESCE((SELECT p2.damage FROM marshal_guard_participants p2 WHERE p2.event_id = e.id ORDER BY p2.damage DESC LIMIT 1), 0) as top_damage
		FROM marshal_guard_events e
		LEFT JOIN marshal_guard_participants p ON p.event_id = e.id
		GROUP BY e.id
		ORDER BY e.event_date DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	events := []MarshalGuardEvent{}
	for rows.Next() {
		var ev MarshalGuardEvent
		var createdByID sql.NullInt64
		if err := rows.Scan(&ev.ID, &ev.EventDate, &ev.TotalAllianceDamage, &ev.Notes,
			&ev.CreatedAt, &createdByID, &ev.ParticipantCount, &ev.TopDamageDealer, &ev.TopDamage); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if createdByID.Valid {
			id := int(createdByID.Int64)
			ev.CreatedByID = &id
		}
		events = append(events, ev)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// GET /api/marshal-guard/{id} — get event with participants
func getMarshalGuardEvent(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	var ev MarshalGuardEvent
	var createdByID sql.NullInt64
	err := db.QueryRow(`SELECT id, event_date, total_alliance_damage, COALESCE(notes, ''), created_at, created_by_id
		FROM marshal_guard_events WHERE id = ?`, id).Scan(
		&ev.ID, &ev.EventDate, &ev.TotalAllianceDamage, &ev.Notes, &ev.CreatedAt, &createdByID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}
	if createdByID.Valid {
		cid := int(createdByID.Int64)
		ev.CreatedByID = &cid
	}

	rows, err := db.Query(`
		SELECT p.id, p.event_id, p.member_id, COALESCE(m.name, ''), p.name_snapshot,
			COALESCE(p.alliance_tag, ''), p.rank_in_event, p.damage, p.attack_count
		FROM marshal_guard_participants p
		LEFT JOIN members m ON m.id = p.member_id
		WHERE p.event_id = ?
		ORDER BY p.rank_in_event`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	ev.Participants = []MarshalGuardParticipant{}
	for rows.Next() {
		var p MarshalGuardParticipant
		var memberID sql.NullInt64
		var attackCount sql.NullInt64
		if err := rows.Scan(&p.ID, &p.EventID, &memberID, &p.MemberName,
			&p.NameSnapshot, &p.AllianceTag, &p.RankInEvent, &p.Damage, &attackCount); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if memberID.Valid {
			mid := int(memberID.Int64)
			p.MemberID = &mid
		}
		if attackCount.Valid {
			ac := int(attackCount.Int64)
			p.AttackCount = &ac
		}
		ev.Participants = append(ev.Participants, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ev)
}

// POST /api/marshal-guard — create event manually (no OCR)
func createMarshalGuardEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventDate           string `json:"event_date"`
		TotalAllianceDamage int64  `json:"total_alliance_damage"`
		Notes               string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.EventDate == "" {
		http.Error(w, "event_date is required", http.StatusBadRequest)
		return
	}

	session, _ := store.Get(r, "session")
	userID, _ := session.Values["user_id"].(int)

	result, err := db.Exec(`INSERT INTO marshal_guard_events (event_date, total_alliance_damage, notes, created_by_id)
		VALUES (?, ?, ?, ?)`, req.EventDate, req.TotalAllianceDamage, req.Notes, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "message": "Event created"})
}

// PUT /api/marshal-guard/{id} — update event metadata
func updateMarshalGuardEvent(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	var req struct {
		EventDate           string `json:"event_date"`
		TotalAllianceDamage int64  `json:"total_alliance_damage"`
		Notes               string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err := db.Exec(`UPDATE marshal_guard_events SET event_date = ?, total_alliance_damage = ?, notes = ? WHERE id = ?`,
		req.EventDate, req.TotalAllianceDamage, req.Notes, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event updated"})
}

// DELETE /api/marshal-guard/{id} — delete event + participants (cascade)
func deleteMarshalGuardEvent(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	// Enable foreign keys for CASCADE
	db.Exec("PRAGMA foreign_keys = ON")
	_, err := db.Exec(`DELETE FROM marshal_guard_events WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event deleted"})
}

// PUT /api/marshal-guard/{id}/participants/{pid} — fix single participant
func updateMarshalGuardParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pid, _ := strconv.Atoi(vars["pid"])
	var req struct {
		MemberID     *int   `json:"member_id"`
		NameSnapshot string `json:"name_snapshot"`
		Damage       int64  `json:"damage"`
		AttackCount  *int   `json:"attack_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err := db.Exec(`UPDATE marshal_guard_participants SET member_id = ?, name_snapshot = ?, damage = ?, attack_count = ? WHERE id = ?`,
		req.MemberID, req.NameSnapshot, req.Damage, req.AttackCount, pid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Participant updated"})
}

// GET /api/marshal-guard/member-stats — per-member MG stats
func getMarshalGuardMemberStats(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT p.member_id, m.name, m.rank,
			COUNT(DISTINCT p.event_id) as event_count,
			COALESCE(SUM(p.damage), 0) as total_damage,
			COALESCE(AVG(p.rank_in_event), 0) as avg_rank,
			COALESCE(MAX(p.damage), 0) as best_damage
		FROM marshal_guard_participants p
		JOIN members m ON m.id = p.member_id
		WHERE p.member_id IS NOT NULL
		GROUP BY p.member_id
		ORDER BY total_damage DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	stats := []MGMemberStats{}
	for rows.Next() {
		var s MGMemberStats
		if err := rows.Scan(&s.MemberID, &s.MemberName, &s.MemberRank,
			&s.EventCount, &s.TotalDamage, &s.AvgRank, &s.BestDamage); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		stats = append(stats, s)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// POST /api/marshal-guard/confirm — atomic create event + participants
func confirmMarshalGuard(w http.ResponseWriter, r *http.Request) {
	var req MGConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.EventDate == "" {
		http.Error(w, "event_date is required", http.StatusBadRequest)
		return
	}

	session, _ := store.Get(r, "session")
	userID, _ := session.Values["user_id"].(int)

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// If overwriting, delete old event first (CASCADE deletes participants)
	if req.OverwriteEventID != nil {
		tx.Exec("PRAGMA foreign_keys = ON")
		if _, err := tx.Exec(`DELETE FROM marshal_guard_events WHERE id = ?`, *req.OverwriteEventID); err != nil {
			http.Error(w, "Failed to overwrite existing event: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	result, err := tx.Exec(`INSERT INTO marshal_guard_events (event_date, total_alliance_damage, notes, created_by_id) VALUES (?, ?, ?, ?)`,
		req.EventDate, req.TotalDamage, req.Notes, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	eventID, _ := result.LastInsertId()

	added := 0
	for _, p := range req.Participants {
		_, err := tx.Exec(`INSERT INTO marshal_guard_participants (event_id, member_id, name_snapshot, alliance_tag, rank_in_event, damage, attack_count)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			eventID, p.MemberID, p.NameSnapshot, p.AllianceTag, p.RankInEvent, p.Damage, p.AttackCount)
		if err != nil {
			log.Printf("MG confirm: failed to insert participant rank %d: %v", p.RankInEvent, err)
			continue
		}
		added++
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event_id": eventID,
		"added":    added,
		"message":  fmt.Sprintf("Event created with %d participants", added),
	})
}

// POST /api/marshal-guard/process-screenshots — OCR parse MG screenshots
func processMarshalGuardScreenshots(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images[]"]
	if len(files) == 0 {
		// Try single-file field name
		files = r.MultipartForm.File["image"]
	}
	if len(files) == 0 {
		http.Error(w, "No images provided", http.StatusBadRequest)
		return
	}
	if len(files) > 40 {
		http.Error(w, "Maximum 40 images allowed", http.StatusBadRequest)
		return
	}

	var allParticipants []MGOCRParticipant
	var eventDate string
	var totalDamage int64

	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			continue
		}
		imageData, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			continue
		}

		// Try row-based extraction first (much more accurate)
		rowParticipants, rowDate, rowTotal := extractMGByRows(imageData)
		if len(rowParticipants) >= 3 {
			log.Printf("MG OCR: row-based extraction found %d participants", len(rowParticipants))
			if rowDate != "" && eventDate == "" {
				eventDate = rowDate
			}
			if rowTotal > totalDamage {
				totalDamage = rowTotal
			}
			for _, p := range rowParticipants {
				merged := false
				for i, existing := range allParticipants {
					if strings.EqualFold(existing.NameSnapshot, p.NameSnapshot) {
						if p.Damage > existing.Damage {
							allParticipants[i] = p
						}
						merged = true
						break
					}
				}
				if !merged {
					allParticipants = append(allParticipants, p)
				}
			}
			continue
		}

		// Mask-based row extraction: the game mail has a FIXED layout
		// Use pattern matching to crop each player row's name + damage regions
		log.Printf("MG OCR: row-based extraction found only %d, using mask-based extraction", len(rowParticipants))
		maskResult := extractMGByMask(imageData)
		if maskResult == nil {
			log.Printf("MG OCR: mask extraction failed, trying raw OCR fallback")
			clientRaw := gosseract.NewClient()
			clientRaw.SetImageFromBytes(imageData)
			textRaw, errRaw := clientRaw.Text()
			clientRaw.Close()
			if errRaw != nil {
				continue
			}
			fallback := parseMarshalGuardText(textRaw)
			maskResult = &fallback
		}

		result := *maskResult
		log.Printf("MG OCR mask: found %d participants", len(result.Participants))

		if result.EventDate != "" && eventDate == "" {
			eventDate = result.EventDate
		}
		if result.TotalDamage > totalDamage {
			totalDamage = result.TotalDamage
		}
		// Merge participants by name using fuzzy matching
		// (OCR may garble names slightly between screenshots)
		for _, p := range result.Participants {
			merged := false
			for i, existing := range allParticipants {
				if strings.EqualFold(existing.NameSnapshot, p.NameSnapshot) ||
					calculateSimilarity(existing.NameSnapshot, p.NameSnapshot) >= 75 {
					if p.Damage > existing.Damage {
						allParticipants[i].Damage = p.Damage
					}
					merged = true
					break
				}
			}
			if !merged {
				allParticipants = append(allParticipants, p)
			}
		}
	}

	// Allow manual event_date override from form field
	if manualDate := r.FormValue("event_date"); manualDate != "" {
		eventDate = manualDate
	}

	// Reassign ranks sequentially after merge: MVP=1, then 2,3,...
	// Sort by damage descending (damage determines rank — higher damage = lower rank number)
	sort.SliceStable(allParticipants, func(i, j int) bool {
		// Both have damage: higher damage ranks first
		if allParticipants[i].Damage > 0 && allParticipants[j].Damage > 0 {
			return allParticipants[i].Damage > allParticipants[j].Damage
		}
		// One has damage, the other doesn't: damage goes first
		if allParticipants[i].Damage > 0 {
			return true
		}
		if allParticipants[j].Damage > 0 {
			return false
		}
		// Neither has damage: keep original order
		return false
	})
	for i := range allParticipants {
		allParticipants[i].RankInEvent = i + 1
	}

	// Match participants to members
	members, err := loadAllMembers()
	if err == nil {
		for i := range allParticipants {
			matchMGParticipant(&allParticipants[i], members)
		}
	}

	// Check for existing event on same date
	var existingEventID *int
	if eventDate != "" {
		var eid int
		err := db.QueryRow(`SELECT id FROM marshal_guard_events WHERE event_date = ?`, eventDate).Scan(&eid)
		if err == nil {
			existingEventID = &eid
		}
	}

	// Sort by rank
	sort.Slice(allParticipants, func(i, j int) bool {
		return allParticipants[i].RankInEvent < allParticipants[j].RankInEvent
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MarshalGuardOCRResult{
		EventDate:       eventDate,
		TotalDamage:     totalDamage,
		Participants:    allParticipants,
		ExistingEventID: existingEventID,
	})
}

// extractMGByRows segments a Marshal Guard screenshot into rows and OCRs each independently.
// The Alliance Exercise ranking layout is:
//
//	| Rank | Avatar | Player Name [Tag] | Damage |
//
// Similar column ratios to VS Points but damage column is on the right.
func extractMGByRows(imageData []byte) ([]MGOCRParticipant, string, int64) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		log.Printf("MG row OCR: failed to decode image: %v", err)
		return nil, "", 0
	}

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	// Analyze screenshot to get data region
	attrs := analyzeScreenshot(img)
	dataRegion := attrs.DataRegion

	// Convert to grayscale for edge detection
	grayFull := convertToGrayscale(img)

	// Find row boundaries using separator line detection
	const minRowH = 30
	rowBounds := findRowBoundaries(grayFull, dataRegion.Top, dataRegion.Bottom, minRowH)
	log.Printf("MG row OCR: found %d rows in data region (image %dx%d, data region %d-%d)",
		len(rowBounds), imgW, imgH, dataRegion.Top, dataRegion.Bottom)

	if len(rowBounds) == 0 {
		return nil, "", 0
	}

	var participants []MGOCRParticipant
	var eventDate string
	var totalDamage int64
	rank := 1

	// First: try to extract date/total from the header area (above data region)
	if dataRegion.Top > 50 {
		headerImg := image.NewRGBA(image.Rect(0, 0, imgW, dataRegion.Top))
		draw.Draw(headerImg, headerImg.Bounds(), img, image.Point{0, 0}, draw.Src)
		var headerBuf bytes.Buffer
		if png.Encode(&headerBuf, headerImg) == nil {
			hClient := gosseract.NewClient()
			hClient.SetImageFromBytes(headerBuf.Bytes())
			headerText, herr := hClient.Text()
			hClient.Close()
			if herr == nil {
				// Extract date
				datePatterns := []*regexp.Regexp{
					regexp.MustCompile(`(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})`),
					regexp.MustCompile(`(\d{1,2})[-/.](\d{1,2})[-/.](\d{4})`),
				}
				for _, line := range strings.Split(headerText, "\n") {
					for pi, pat := range datePatterns {
						if m := pat.FindStringSubmatch(line); m != nil {
							var y, mo, d int
							if pi == 0 {
								y, _ = strconv.Atoi(m[1])
								mo, _ = strconv.Atoi(m[2])
								d, _ = strconv.Atoi(m[3])
							} else {
								d, _ = strconv.Atoi(m[1])
								mo, _ = strconv.Atoi(m[2])
								y, _ = strconv.Atoi(m[3])
							}
							if y >= 2024 && y <= 2030 && mo >= 1 && mo <= 12 && d >= 1 && d <= 31 {
								eventDate = fmt.Sprintf("%04d-%02d-%02d", y, mo, d)
							}
						}
					}
				}
				// Extract total damage from header
				totalAbbrRe := regexp.MustCompile(`(?i)(\d+[,.]?\d*)\s*([GMK])`)
				for _, line := range strings.Split(headerText, "\n") {
					lower := strings.ToLower(line)
					if strings.Contains(lower, "total") || strings.Contains(lower, "alliance") || strings.Contains(lower, "score") || strings.Contains(lower, "damage") {
						if m := totalAbbrRe.FindStringSubmatch(line); m != nil {
							totalDamage = parseDamageValue(strings.ReplaceAll(m[1], ",", ""), m[2])
						}
					}
				}
				log.Printf("MG row OCR header: date=%q total=%d headerText=%q", eventDate, totalDamage, headerText)
			}
		}
	}

	// Process each data row
	for i, rb := range rowBounds {
		rowTop, rowBottom := rb[0], rb[1]
		rowH := rowBottom - rowTop
		if rowH < minRowH {
			continue
		}

		// Layout: | rank/avatar (0-28%) | name (28%-65%) | damage (65%-100%) |
		// Detect avatar boundary
		nameStartX := detectAvatarEndX(grayFull, rowTop, rowBottom, imgW*35/100)
		if nameStartX < imgW*15/100 {
			nameStartX = imgW * 20 / 100 // safety minimum
		}

		nameEndX := imgW * 65 / 100
		damageStartX := imgW * 65 / 100

		nameTopH := rowH * 60 / 100 // top 60% has the name

		// Extract name region
		nameImg := image.NewRGBA(image.Rect(0, 0, nameEndX-nameStartX, nameTopH))
		draw.Draw(nameImg, nameImg.Bounds(), img, image.Point{nameStartX, rowTop}, draw.Src)

		// Extract damage region
		damageImg := image.NewRGBA(image.Rect(0, 0, imgW-damageStartX, rowH))
		draw.Draw(damageImg, damageImg.Bounds(), img, image.Point{damageStartX, rowTop}, draw.Src)

		// Scale up for better OCR
		scaledName := scaleImage(nameImg, 3)
		grayName := convertToGrayscale(scaledName)

		scaledDamage := scaleImage(damageImg, 3)
		grayDamage := convertToGrayscale(scaledDamage)

		// OCR name
		var nameBuf bytes.Buffer
		if err := png.Encode(&nameBuf, grayName); err != nil {
			continue
		}
		nameClient := gosseract.NewClient()
		nameClient.SetImageFromBytes(nameBuf.Bytes())
		nameClient.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
		nameText, err := nameClient.Text()
		nameClient.Close()
		if err != nil || len(strings.TrimSpace(nameText)) == 0 {
			continue
		}

		// OCR damage — allow digits, commas, dots, G/M/K
		var dmgBuf bytes.Buffer
		if err := png.Encode(&dmgBuf, grayDamage); err != nil {
			continue
		}
		dmgClient := gosseract.NewClient()
		dmgClient.SetImageFromBytes(dmgBuf.Bytes())
		dmgClient.SetVariable("tessedit_char_whitelist", "0123456789,.GMKgmk")
		dmgClient.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
		dmgText, err := dmgClient.Text()
		dmgClient.Close()
		if err != nil || len(strings.TrimSpace(dmgText)) == 0 {
			log.Printf("MG row %d: name=%q but no damage text", i+1, strings.TrimSpace(nameText))
			continue
		}

		// Parse damage value
		dmgStr := strings.TrimSpace(dmgText)
		var damage int64
		abbrRe := regexp.MustCompile(`(\d+[.,]?\d*)\s*([GMKgmk])`)
		if m := abbrRe.FindStringSubmatch(dmgStr); m != nil {
			damage = parseDamageValue(strings.ReplaceAll(m[1], ",", "."), strings.ToUpper(m[2]))
		} else {
			// Plain number
			numStr := regexp.MustCompile(`[^0-9]`).ReplaceAllString(dmgStr, "")
			if v, err := strconv.ParseInt(numStr, 10, 64); err == nil && v >= 100000 {
				damage = v
			}
		}

		if damage <= 0 {
			log.Printf("MG row %d: name=%q damage text=%q -> unparseable", i+1, strings.TrimSpace(nameText), dmgStr)
			continue
		}

		// Clean name
		name := cleanPlayerName(strings.TrimSpace(nameText))

		// Extract alliance tag from name
		var tag string
		tagRe := regexp.MustCompile(`\[([^\]]+)\]`)
		if m := tagRe.FindStringSubmatch(name); m != nil {
			tag = m[1]
			name = strings.TrimSpace(strings.Replace(name, m[0], "", 1))
		}

		if len(name) < 2 {
			continue
		}

		log.Printf("MG row %d: rank=%d name=%q tag=%q damage=%d (raw: %q)", i+1, rank, name, tag, damage, dmgStr)

		participants = append(participants, MGOCRParticipant{
			RankInEvent:  rank,
			NameSnapshot: name,
			AllianceTag:  tag,
			Damage:       damage,
		})
		rank++
	}

	return participants, eventDate, totalDamage
}

func loadDeletedMembers() ([]Member, error) {
	rows, err := db.Query(`SELECT id, name, rank, nickname FROM members WHERE deleted_at IS NOT NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []Member
	for rows.Next() {
		var m Member
		var nickname sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &m.Rank, &nickname); err != nil {
			continue
		}
		if nickname.Valid {
			m.Nickname = &nickname.String
		}
		members = append(members, m)
	}
	return members, nil
}

func loadAllMembers() ([]Member, error) {
	rows, err := db.Query(`SELECT id, name, rank, nickname FROM members WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []Member
	for rows.Next() {
		var m Member
		var nickname sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &m.Rank, &nickname); err != nil {
			continue
		}
		if nickname.Valid {
			m.Nickname = &nickname.String
		}
		members = append(members, m)
	}
	return members, nil
}

// parsePlayerTag extracts the alliance tag and plain player name from strings like
// "[RSRP]Gargoland" or "[RSRPlJazzyopolis" (where ] was OCR'd as l, 1, | or I).
// knownTag (from settings) is used for fuzzy bracket matching before standard parsing.
func parsePlayerTag(name, knownTag string) (tag, nameOnly string) {
	name = strings.TrimSpace(name)
	// Prefer known-tag match with fuzzy closing bracket.
	if knownTag != "" {
		upper := strings.ToUpper(name)
		prefix := "[" + strings.ToUpper(knownTag)
		if idx := strings.Index(upper, prefix); idx >= 0 {
			after := name[idx+len(prefix):]
			if len(after) > 0 && strings.ContainsRune("]l1|I", rune(after[0])) {
				return knownTag, strings.TrimSpace(after[1:])
			}
		}
	}
	// Standard parsing: first [...] group.
	if close := strings.Index(name, "]"); close >= 0 {
		open := strings.Index(name, "[")
		if open >= 0 && open < close {
			return name[open+1 : close], strings.TrimSpace(name[close+1:])
		}
	}
	return "", name
}

// mgOcrNormForCompare maps visually-similar characters to a canonical form for
// name-matching only, so OCR confusions (O↔0, l/I↔1) do not prevent a fuzzy match.
func mgOcrNormForCompare(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("o", "0", "l", "1", "i", "1").Replace(s)
	return s
}

// mgSimilarityNorm computes the Levenshtein-based similarity (0–100) between two
// already-normalised strings.
func mgSimilarityNorm(n1, n2 string) int {
	if n1 == n2 {
		return 100
	}
	dist := levenshteinDistance(n1, n2)
	maxLen := len(n1)
	if len(n2) > maxLen {
		maxLen = len(n2)
	}
	if maxLen == 0 {
		return 0
	}
	return (maxLen - dist) * 100 / maxLen
}

func matchMGParticipant(p *MGOCRParticipant, members []Member) {
	name := strings.TrimSpace(p.NameSnapshot)
	if name == "" {
		return
	}
	lower := strings.ToLower(name)

	// Exact name match
	for _, m := range members {
		if strings.ToLower(m.Name) == lower {
			p.MemberID = &m.ID
			p.MemberName = m.Name
			return
		}
	}
	// Nickname match
	for _, m := range members {
		if m.Nickname != nil && strings.ToLower(*m.Nickname) == lower {
			p.MemberID = &m.ID
			p.MemberName = m.Name
			return
		}
	}
	// Fuzzy match (≥70% similarity)
	bestScore := 0
	bestIdx := -1
	for i, m := range members {
		sim := calculateSimilarity(name, m.Name)
		if m.Nickname != nil && *m.Nickname != "" {
			nickSim := calculateSimilarity(name, *m.Nickname)
			if nickSim > sim {
				sim = nickSim
			}
		}
		if sim > bestScore {
			bestScore = sim
			bestIdx = i
		}
	}
	if bestScore >= 70 && bestIdx >= 0 {
		p.MemberID = &members[bestIdx].ID
		p.MemberName = members[bestIdx].Name
		return
	}

	// Second pass: OCR character normalisation (O↔0, l/I↔1, try without
	// spurious leading character added by OCR e.g. "J" before "KM011").
	ocrNorm := mgOcrNormForCompare(name)
	bestScore = 0
	bestIdx = -1
	for i, m := range members {
		dbNorm := mgOcrNormForCompare(m.Name)
		sim := mgSimilarityNorm(ocrNorm, dbNorm)
		// Also try without the first character (spurious leading OCR char).
		if len(ocrNorm) > 3 {
			if s2 := mgSimilarityNorm(ocrNorm[1:], dbNorm); s2 > sim {
				sim = s2
			}
		}
		if m.Nickname != nil && *m.Nickname != "" {
			nickNorm := mgOcrNormForCompare(*m.Nickname)
			if s := mgSimilarityNorm(ocrNorm, nickNorm); s > sim {
				sim = s
			}
		}
		if sim > bestScore {
			bestScore = sim
			bestIdx = i
		}
	}
	if bestScore >= 70 && bestIdx >= 0 {
		p.MemberID = &members[bestIdx].ID
		p.MemberName = members[bestIdx].Name
	}
}

// extractMGByMask uses a fixed proportional layout mask to extract player data from
// Marshal Guard mail screenshots. The game uses a stable layout across devices:
//   - MVP section: top ~32% of image
//   - Player list: 5 rows in the middle band
//   - Each row: avatar on left, [TAG]Name on top text line, "Total Damage: X.XXG" below
//
// Key technique: each row is cropped into TWO separate sub-regions
//  1. Name region (top half of row)             → OCR with letter whitelist
//  2. Damage VALUE region (bottom half, right portion → label is cut off!)
//     → OCR with digit-only whitelist
//
// Cutting off the "Total Damage:" label eliminates the leading "1" garble (from
// the trailing "l" of "Totall") at the source. Whitelisted OCR forbids any
// letter→digit substitution. The result is dramatically cleaner than block OCR.
func extractMGByMask(imageData []byte) *MarshalGuardOCRResult {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		log.Printf("MG mask: decode error: %v", err)
		return nil
	}

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	// All layout positions are proportional (% of image dimensions)
	// Works for any device resolution since the game mail uses fixed proportions.
	// Player rows use text-line detection instead of fixed proportions so they
	// remain accurate even when a status bar or notification shifts the layout.
	const (
		// Horizontal: text region (right of avatar and rank badge)
		nameXStart = 0.30 // right of avatar frame
		nameXEnd   = 0.78
		// Damage VALUE crop excludes the "Total Damage:" label:
		// the label takes the left ~half of the damage line; value sits on the right.
		dmgXStart = 0.55 // skip "Total Damage:" label
		dmgXEnd   = 0.82

		// MVP section (top of mail, larger card with combat-power number and damage line)
		mvpNameYStart = 0.298 // line containing "[RSRP]Gargoland"
		mvpNameYEnd   = 0.327 // tight: avoid the 93,943,806 line below
		mvpDmgYStart  = 0.385 // "Total Damage: X.XXG" line under "Attacks: N"
		mvpDmgYEnd    = 0.420
		mvpXStart     = 0.43 // right of the avatar AND the shield icon
		mvpXEnd       = 0.85
		mvpDmgXStart  = 0.46 // skip "Total Damage:" label on MVP line

		// Date line ("2026-5-12 20:30:05") near the bottom of mail body
		dateYStart = 0.88
		dateYEnd   = 0.91
	)

	pxX := func(p float64) int { return int(float64(imgW) * p) }
	pxY := func(p float64) int { return int(float64(imgH) * p) }

	result := &MarshalGuardOCRResult{}
	var participants []MGOCRParticipant

	// Debug overlay setup (no-op unless MG_DEBUG_DIR env var is set)
	mgDebugInit(img)
	defer mgDebugSaveOverlay(img)

	// --- MVP ---
	mvpNameText := ocrCropField(img,
		pxX(mvpXStart), pxY(mvpNameYStart),
		pxX(mvpXEnd), pxY(mvpNameYEnd),
		mgFieldName, "mvp_name")
	mvpDmgText := ocrCropField(img,
		pxX(mvpDmgXStart), pxY(mvpDmgYStart),
		pxX(mvpXEnd), pxY(mvpDmgYEnd),
		mgFieldDamage, "mvp_dmg")
	log.Printf("MG mask MVP: name=%q damage=%q", mvpNameText, mvpDmgText)

	mvpParsedName, mvpTag := parseMGMaskName(mvpNameText)
	mvpDamage := parseMGMaskDamage(mvpDmgText)
	if mvpParsedName != "" {
		participants = append(participants, MGOCRParticipant{
			RankInEvent:  1,
			NameSnapshot: mvpParsedName,
			AllianceTag:  mvpTag,
			Damage:       mvpDamage,
		})
		if mvpDamage > result.TotalDamage {
			result.TotalDamage = mvpDamage
		}
	}

	// --- Date ---
	// Date text is centered in the light content area, so crop only the center
	// to avoid dark side margins and navigation area that confuse binarization.
	dateText := ocrCropField(img, pxX(0.15), pxY(dateYStart), pxX(0.85), pxY(dateYEnd), mgFieldDate, "date")
	dateRe := regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})`)
	if dm := dateRe.FindStringSubmatch(dateText); dm != nil {
		y, _ := strconv.Atoi(dm[1])
		m, _ := strconv.Atoi(dm[2])
		d, _ := strconv.Atoi(dm[3])
		result.EventDate = fmt.Sprintf("%04d-%02d-%02d", y, m, d)
	}

	// --- Player rows ---
	// Detect text lines in the body strip (below "Damage Ranking" header, above date).
	// Each player card has exactly 2 lines: name on top, "Total Damage: X" below.
	// Five cards = ten lines. Detection makes row positioning resolution-invariant
	// and robust against vertical shifts (notification banners, status bars, etc.).
	// Detect text lines in the body strip. Start below "Damage Ranking 2-20"
	// header (at ~44%) and end before date (at ~90%).
	bodyTop := pxY(0.47)
	bodyBot := pxY(0.86) // stop before date line and nav bar divider
	lines := detectMGTextLines(img, pxX(0.30), pxX(0.78), bodyTop, bodyBot)

	// Debug: render detected text lines as yellow boxes in the overlay
	if mgDebug != nil {
		for i, ln := range lines {
			mgDebug.boxes = append(mgDebug.boxes, mgDebugBox{
				label: fmt.Sprintf("L%d", i),
				x1:    pxX(0.28), y1: ln[0],
				x2: pxX(0.80), y2: ln[1],
				color: color.RGBA{255, 220, 0, 255}, // yellow
			})
		}
	}
	log.Printf("MG mask: detected %d text lines in body strip", len(lines))

	// Strategy: edge detection reliably finds DAMAGE lines ("Total Damage: X.XXG")
	// because they have high edge density. Player name lines have lower density and
	// often don't produce a signal. We pick the 5 strongest/tallest lines as damage
	// lines, then compute each name position as a fixed offset ABOVE the damage line.
	//
	// In the game layout (from visual inspection of the debug overlay):
	//   name:   ~30px above the damage line
	//   damage: detected line
	//   gap:    space to next card

	// Filter: keep lines with height between 20-80px (text lines, not artifacts)
	var textLines [][2]int
	for _, ln := range lines {
		h := ln[1] - ln[0]
		if h >= 20 && h <= 80 {
			textLines = append(textLines, ln)
		}
	}

	// From the remaining lines, group those that are within 80px of each other
	// into clusters. For each cluster, pick the LOWER one (higher Y = damage line).
	var dmgLines [][2]int
	for i := 0; i < len(textLines); i++ {
		if i+1 < len(textLines) && textLines[i+1][0]-textLines[i][0] < 80 {
			dmgLines = append(dmgLines, textLines[i+1])
			i++ // skip the paired name line
		} else {
			dmgLines = append(dmgLines, textLines[i])
		}
	}
	if len(dmgLines) > 5 {
		dmgLines = dmgLines[len(dmgLines)-5:]
	}
	log.Printf("MG mask: using %d damage lines for name-offset extraction", len(dmgLines))

	var playerRows []MGOCRParticipant
	if len(dmgLines) >= 3 {
		// Compute name offset: name line sits one line-height above the damage line
		// top, with a small gap between them (~5-8px in 2048-tall images).
		lineH := dmgLines[0][1] - dmgLines[0][0]

		// --- Overlap-based boundary detection ---
		// Crop all name and damage rows at FULL card width, compare pairwise pixel
		// differences per column to find where the consistent elements (alliance tag
		// "[RSRP]" and "Total Damage:" label) end and variable content begins.
		// This automatically adapts to any tag length or font size.
		const (
			wideNameXStart = 0.15 // full card width for comparison
			wideNameXEnd   = 0.78
		)

		pad := 4
		var nameRects [][4]int
		for _, dmgLine := range dmgLines {
			nameBot := dmgLine[0] - 5
			nameTop := nameBot - lineH
			if nameTop < bodyTop {
				nameTop = bodyTop
			}
			if nameBot-nameTop < 20 {
				continue
			}
			nameRects = append(nameRects, [4]int{pxX(wideNameXStart), nameTop - pad, pxX(wideNameXEnd), nameBot + pad})
		}

		// Find where variable content starts using overlap comparison
		nameBoundary := mgFindContentStart(img, nameRects)
		// Damage boundary is less reliable (varies per shot), use fixed proportion
		// since "Total Damage:" label has consistent width across all cards.

		// Convert boundary offsets to absolute X coordinates
		nameXActual := pxX(wideNameXStart) + nameBoundary
		dmgXActual := pxX(dmgXStart) // use fixed proportion for damage

		// Back off from the detected boundary to INCLUDE the alliance tag.
		// parseMGMaskName will strip it. The tag "[RSRP]" is ~80px wide.
		// This ensures the full tag+name is captured regardless of detection precision.
		if nameBoundary > 0 {
			nameXActual -= 80
			// Don't go further left than the fixed fallback
			if nameXActual < pxX(nameXStart) {
				nameXActual = pxX(nameXStart)
			}
		}

		// Fallback to fixed proportions if boundary detection fails
		if nameBoundary == 0 {
			nameXActual = pxX(nameXStart)
		}

		log.Printf("MG mask: nameX=%d (%.2f), dmgX=%d (%.2f), nameBoundary=%d",
			nameXActual, float64(nameXActual)/float64(imgW),
			dmgXActual, float64(dmgXActual)/float64(imgW), nameBoundary)

		// Now OCR each row using the detected boundaries
		rowIdx := 0
		for _, dmgLine := range dmgLines {
			nameBot := dmgLine[0] - 5
			nameTop := nameBot - lineH
			if nameTop < bodyTop {
				nameTop = bodyTop
			}
			if nameBot-nameTop < 20 {
				continue
			}

			nameText := ocrCropField(img,
				nameXActual, nameTop-pad,
				pxX(nameXEnd), nameBot+pad,
				mgFieldName, fmt.Sprintf("row%d_name", rowIdx))
			dmgText := ocrCropField(img,
				dmgXActual, dmgLine[0]-pad,
				pxX(dmgXEnd), dmgLine[1]+pad,
				mgFieldDamage, fmt.Sprintf("row%d_dmg", rowIdx))
			log.Printf("MG mask row %d (dmg@y=%d-%d, name@y=%d-%d): name=%q damage=%q",
				rowIdx, dmgLine[0], dmgLine[1], nameTop, nameBot, nameText, dmgText)
			name, tag := parseMGMaskName(nameText)
			damage := parseMGMaskDamage(dmgText)
			if name == "" {
				rowIdx++
				continue
			}
			if mvpParsedName != "" && strings.EqualFold(name, mvpParsedName) {
				rowIdx++
				continue
			}
			playerRows = append(playerRows, MGOCRParticipant{
				RankInEvent:  len(playerRows) + 2,
				NameSnapshot: name,
				AllianceTag:  tag,
				Damage:       damage,
			})
			rowIdx++
		}
	}

	// Safety net: enforce descending damage order across player rows
	// (rare with whitelisted OCR, but handles any remaining garble)
	if len(playerRows) > 1 {
		mgEnforceMonotonicity(playerRows)
	}
	participants = append(participants, playerRows...)

	result.Participants = participants
	return result
}

// mgFieldKind selects per-field OCR config (whitelist + page segmentation mode)
type mgFieldKind int

const (
	mgFieldDamage mgFieldKind = iota // digits + . , G M K
	mgFieldName                      // letters + digits + [ ] _ space
	mgFieldDate                      // digits + -
)

// mgFindContentStart overlaps multiple card row crops and finds the X pixel
// where variable content (player name or damage number) begins. The consistent
// region (alliance tag "[RSRP]" or "Total Damage:" label) has low pairwise
// pixel differences across cards; the variable region has high differences.
// rects: each entry is [x1, y1, x2, y2] defining a crop rectangle on img.
// Returns X offset relative to crop left edge, or 0 if detection fails.
func mgFindContentStart(img image.Image, rects [][4]int) int {
	if len(rects) < 3 {
		return 0
	}
	// Determine common width and height
	minW := rects[0][2] - rects[0][0]
	minH := rects[0][3] - rects[0][1]
	for _, r := range rects[1:] {
		w := r[2] - r[0]
		h := r[3] - r[1]
		if w < minW {
			minW = w
		}
		if h < minH {
			minH = h
		}
	}
	if minW < 30 || minH < 10 {
		return 0
	}

	// For each column x, compare pixel values at EVERY row (step 2-3px) across all pairs.
	// In the tag/label region, all cards have identical text → same pixels → low diff.
	// In the name/number region, different text → different pixels → high diff.
	n := len(rects)
	yStep := 3
	if minH < 20 {
		yStep = 2
	}

	colDiff := make([]float64, minW)
	for x := 0; x < minW; x++ {
		var totalDiff float64
		pairs := 0
		for i := 0; i < n-1; i++ {
			for j := i + 1; j < n; j++ {
				ri := rects[i]
				rj := rects[j]
				for dy := 0; dy < minH; dy += yStep {
					yi := ri[1] + dy
					yj := rj[1] + dy
					xi := ri[0] + x
					xj := rj[0] + x

					ri8, gi8, bi8, _ := img.At(xi, yi).RGBA()
					rj8, gj8, bj8, _ := img.At(xj, yj).RGBA()

					minI := uint8(ri8 >> 8)
					if g := uint8(gi8 >> 8); g < minI {
						minI = g
					}
					if b := uint8(bi8 >> 8); b < minI {
						minI = b
					}
					minJ := uint8(rj8 >> 8)
					if g := uint8(gj8 >> 8); g < minJ {
						minJ = g
					}
					if b := uint8(bj8 >> 8); b < minJ {
						minJ = b
					}

					diff := int(minI) - int(minJ)
					if diff < 0 {
						diff = -diff
					}
					totalDiff += float64(diff)
					pairs++
				}
			}
		}
		if pairs > 0 {
			colDiff[x] = totalDiff / float64(pairs)
		}
	}

	// Smooth with a 10px window
	smoothed := make([]float64, minW)
	win := 10
	for x := 0; x < minW; x++ {
		start := x - win/2
		if start < 0 {
			start = 0
		}
		end := x + win/2
		if end > minW {
			end = minW
		}
		var s float64
		for i := start; i < end; i++ {
			s += colDiff[i]
		}
		smoothed[x] = s / float64(end-start)
	}

	// Find the LOW-difference island (consistent tag or label) and return its right edge.
	// There may be multiple low regions (card backgrounds). We want the RIGHTMOST low
	// region in the middle portion that's followed by a high-diff region (variable content).
	maxVal := 0.0
	for _, v := range smoothed {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal < 3 {
		log.Printf("MG mask boundary: maxVal=%.1f too low, giving up", maxVal)
		return 0
	}

	// Threshold: anything below 35% of max is "low" (consistent)
	lowThresh := maxVal * 0.35

	// Find all valleys (low regions) and their right edges
	// Scan left-to-right, track when we enter/exit a low region
	type valley struct{ left, right int }
	var valleys []valley
	inLow := false
	lowStart := 0
	for x := 0; x < minW; x++ {
		if smoothed[x] < lowThresh {
			if !inLow {
				lowStart = x
				inLow = true
			}
		} else {
			if inLow {
				valleys = append(valleys, valley{lowStart, x})
				inLow = false
			}
		}
	}
	if inLow {
		valleys = append(valleys, valley{lowStart, minW - 1})
	}

	// Find the RIGHTMOST valley whose right edge is between 20% and 65% of width.
	// This is the tag/label region, and the name/number starts at its right edge.
	boundary := 0
	for i := len(valleys) - 1; i >= 0; i-- {
		v := valleys[i]
		rightEdge := v.right
		// Valley must be at least 15px wide (to be actual text, not noise)
		if v.right-v.left < 15 {
			continue
		}
		if rightEdge >= minW*20/100 && rightEdge <= minW*65/100 {
			boundary = rightEdge
			break
		}
	}

	if boundary == 0 {
		log.Printf("MG mask boundary: no valid valley found, maxVal=%.1f, valleys=%d",
			maxVal, len(valleys))
		return 0
	}

	log.Printf("MG mask boundary: width=%d, contentStart=%d (%.0f%%), maxDiff=%.1f, valleys=%d",
		minW, boundary, float64(boundary)/float64(minW)*100, maxVal, len(valleys))
	return boundary
}

// autoDetectInvert samples the center content area of an RGBA crop.
// Returns true if the majority of the interior is dark (light text on dark bg).
// Returns false if the majority is light (dark text on light bg → already correct).
// Uses a 25% inset sampling grid to avoid border artifacts and decorations.
func autoDetectInvert(cropped *image.RGBA) bool {
	bounds := cropped.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w < 8 || h < 8 {
		return false // default: don't invert (assume dark-on-light)
	}
	// Use histogram mode (most frequent luminance bucket) to detect background.
	// Background pixels always outnumber text pixels in a text crop, so the mode
	// represents the background color regardless of crop size or text thickness.
	var hist [256]int
	stepX := w / 40
	if stepX < 1 {
		stepX = 1
	}
	stepY := h / 20
	if stepY < 1 {
		stepY = 1
	}
	for y := 0; y < h; y += stepY {
		for x := 0; x < w; x += stepX {
			r, g, b, _ := cropped.At(x, y).RGBA()
			lum := uint8((299*r + 587*g + 114*b) / 1000 >> 8)
			hist[lum]++
		}
	}
	// Find mode (bucket with highest count)
	modeVal := 0
	modeCount := 0
	for i, c := range hist {
		if c > modeCount {
			modeCount = c
			modeVal = i
		}
	}
	return modeVal < 128 // dark background → light text → invert needed
}

// preprocessForOCR converts an RGBA crop into an OCR-ready grayscale image:
//  1. Convert to grayscale (luminance)
//  2. Auto-detect text polarity (invert if dark background)
//  3. Otsu binarize (auto-threshold) — pure black text on pure white
//  4. Add white border padding (Tesseract docs recommend ≥10px)
//  5. Upscale so that capital letter height is ~36px (Tesseract sweet spot)
//
// All steps use proportional / threshold-based logic so the function works
// equally well on small or large device screenshots.
func preprocessForOCR(cropped *image.RGBA) *image.Gray {
	bounds := cropped.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return image.NewGray(image.Rect(0, 0, 1, 1))
	}

	// Step 1+2: grayscale using min-channel (better for colored text on colored
	// backgrounds — e.g. red text on pink: min(R,G,B) of text ≈ 0 vs background ≈ 170).
	// Then optionally invert for light-on-dark cases.
	invert := autoDetectInvert(cropped)
	gray := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := cropped.At(x, y).RGBA()
			// min-channel: gives lowest component, maximizes contrast for colored text
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			minCh := r8
			if g8 < minCh {
				minCh = g8
			}
			if b8 < minCh {
				minCh = b8
			}
			if invert {
				minCh = 255 - minCh
			}
			gray.SetGray(x, y, color.Gray{Y: minCh})
		}
	}

	// Step 3: Otsu binarize
	thr := computeOtsuThreshold(gray)
	binary := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if gray.GrayAt(x, y).Y < thr {
				binary.SetGray(x, y, color.Gray{Y: 0}) // text
			} else {
				binary.SetGray(x, y, color.Gray{Y: 255}) // background
			}
		}
	}

	// Step 4: add white border padding (proportional to crop size, min 10px)
	pad := h / 4
	if pad < 10 {
		pad = 10
	}
	if pad > 40 {
		pad = 40
	}
	paddedW := w + 2*pad
	paddedH := h + 2*pad
	padded := image.NewGray(image.Rect(0, 0, paddedW, paddedH))
	// Fill with white
	for i := range padded.Pix {
		padded.Pix[i] = 255
	}
	// Copy binary into center
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			padded.SetGray(x+pad, y+pad, binary.GrayAt(x, y))
		}
	}

	// Step 5: upscale so cap-height is ~48px (Tesseract optimum for small chars)
	// Cap height is roughly 60% of text row height. For a single-line crop,
	// h ≈ cap-height / 0.6. Target padded height to give cap-height ≈ 48.
	targetCapHeight := 48
	estimatedCapHeight := int(float64(h) * 0.6)
	if estimatedCapHeight < 1 {
		estimatedCapHeight = 1
	}
	scale := targetCapHeight / estimatedCapHeight
	if scale < 2 {
		scale = 2
	}
	if scale > 6 {
		scale = 6
	}
	scaledW := paddedW * scale
	scaledH := paddedH * scale
	scaled := image.NewGray(image.Rect(0, 0, scaledW, scaledH))
	for y := 0; y < scaledH; y++ {
		for x := 0; x < scaledW; x++ {
			scaled.SetGray(x, y, padded.GrayAt(x/scale, y/scale))
		}
	}
	return scaled
}

// ocrCropField crops a region, applies the full preprocessing pipeline,
// and runs OCR with field-specific Tesseract config (PSM + character whitelist).
// fieldKind controls which whitelist and PSM mode is used.
// mgDebugCtx holds per-extraction debug state. Created when MG_DEBUG_DIR is set.
type mgDebugContext struct {
	dir    string
	prefix string       // e.g. "screenshot3"
	boxes  []mgDebugBox // collected for the overlay
}

type mgDebugBox struct {
	label          string
	x1, y1, x2, y2 int
	color          color.RGBA
}

// mgDebug is set per-call to extractMGByMask. nil = debug disabled.
var mgDebug *mgDebugContext
var mgDebugSeq int

func mgDebugInit(img image.Image) {
	dir := os.Getenv("MG_DEBUG_DIR")
	if dir == "" {
		mgDebug = nil
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	mgDebugSeq++
	mgDebug = &mgDebugContext{
		dir:    dir,
		prefix: fmt.Sprintf("shot%02d", mgDebugSeq),
	}
}

func mgDebugSaveCrop(label string, raw *image.RGBA, processed *image.Gray, x1, y1, x2, y2 int, col color.RGBA) {
	if mgDebug == nil {
		return
	}
	mgDebug.boxes = append(mgDebug.boxes, mgDebugBox{label, x1, y1, x2, y2, col})
	base := filepath.Join(mgDebug.dir, fmt.Sprintf("%s_%s", mgDebug.prefix, label))
	if f, err := os.Create(base + "_raw.png"); err == nil {
		_ = png.Encode(f, raw)
		_ = f.Close()
	}
	if f, err := os.Create(base + "_proc.png"); err == nil {
		_ = png.Encode(f, processed)
		_ = f.Close()
	}
}

// mgDebugSaveOverlay draws all collected crop boxes onto a copy of the
// source image and writes it as {prefix}_overlay.png. Helps visually
// verify whether the proportional crop coordinates land where we think.
func mgDebugSaveOverlay(src image.Image) {
	if mgDebug == nil || len(mgDebug.boxes) == 0 {
		return
	}
	b := src.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	for _, box := range mgDebug.boxes {
		drawRect(out, box.x1, box.y1, box.x2, box.y2, box.color, 3)
	}
	f, err := os.Create(filepath.Join(mgDebug.dir, mgDebug.prefix+"_overlay.png"))
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, out)
}

func drawRect(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		for x := x1; x < x2; x++ {
			img.SetRGBA(x, y1+t, col)
			img.SetRGBA(x, y2-1-t, col)
		}
		for y := y1; y < y2; y++ {
			img.SetRGBA(x1+t, y, col)
			img.SetRGBA(x2-1-t, y, col)
		}
	}
}

// detectMGTextLines finds vertical extents of bright text lines in a vertical strip.
// Algorithm: count high-luminance pixels per row (Y-projection), smooth with a 5-row
// box filter, threshold at ~3% of strip width, return runs that are at least 8px tall.
//
// The game UI uses light text on dark/medium backgrounds, so high-luminance pixels are
// reliable text markers. Each text line shows up as a clear plateau in the projection;
// vertical gaps between rows produce near-zero counts.
//
// This makes player-row positions resolution-invariant and resilient to vertical
// shifts (status bar, notifications, different mail header heights).
func detectMGTextLines(img image.Image, xStart, xEnd, yStart, yEnd int) [][2]int {
	bounds := img.Bounds()
	if xStart < bounds.Min.X {
		xStart = bounds.Min.X
	}
	if xEnd > bounds.Max.X {
		xEnd = bounds.Max.X
	}
	if yStart < bounds.Min.Y {
		yStart = bounds.Min.Y
	}
	if yEnd > bounds.Max.Y {
		yEnd = bounds.Max.Y
	}
	stripW := xEnd - xStart
	stripH := yEnd - yStart
	if stripW <= 0 || stripH <= 0 {
		return nil
	}

	// Per-row count of horizontal-edge pixels.
	// Text strokes produce sharp brightness transitions; smooth card/page
	// backgrounds do not. We count pixels where the luminance change over a
	// 3-pixel horizontal window exceeds 40 (out of 255). This signal cleanly
	// isolates text lines regardless of the underlying panel/page color.
	counts := make([]int, stripH)
	lumAt := func(x, y int) int {
		r, g, b, _ := img.At(x, y).RGBA()
		return (299*int(r>>8) + 587*int(g>>8) + 114*int(b>>8)) / 1000
	}
	for yi := 0; yi < stripH; yi++ {
		y := yStart + yi
		c := 0
		for x := xStart; x < xEnd-3; x++ {
			d := lumAt(x+3, y) - lumAt(x, y)
			if d < 0 {
				d = -d
			}
			if d > 40 {
				c++
			}
		}
		counts[yi] = c
	}

	// 3-tap box smoother — bridges 2-3px intra-glyph gaps but not inter-line gaps.
	sm := make([]int, stripH)
	for i := 0; i < stripH; i++ {
		s, n := 0, 0
		for k := -1; k <= 1; k++ {
			j := i + k
			if j >= 0 && j < stripH {
				s += counts[j]
				n++
			}
		}
		sm[i] = s / n
	}

	// Threshold: ~4% of strip width. Need to be low enough to detect both
	// player name lines (fewer characters, less edge density) AND damage lines
	// (more text, higher edge density). 4% catches names with ~15+ edge pixels.
	threshold := stripW / 25
	if threshold < 4 {
		threshold = 4
	}

	var runs [][2]int
	inRun := false
	runStart := 0
	for i := 0; i < stripH; i++ {
		if sm[i] >= threshold {
			if !inRun {
				inRun = true
				runStart = i
			}
		} else if inRun {
			if i-runStart >= 8 {
				runs = append(runs, [2]int{yStart + runStart, yStart + i})
			}
			inRun = false
		}
	}
	if inRun && stripH-runStart >= 8 {
		runs = append(runs, [2]int{yStart + runStart, yStart + stripH})
	}

	// Post-process: if any run is much taller than a typical line (>1.6× the
	// median height), split it at the local minimum of the smoothed signal.
	// Two adjacent text lines can merge when the gap between them is small.
	if len(runs) >= 3 {
		heights := make([]int, len(runs))
		for i, r := range runs {
			heights[i] = r[1] - r[0]
		}
		sorted := append([]int(nil), heights...)
		sort.Ints(sorted)
		medH := sorted[len(sorted)/2]
		maxH := medH * 16 / 10 // 1.6×

		var split [][2]int
		for _, r := range runs {
			if r[1]-r[0] <= maxH {
				split = append(split, r)
				continue
			}
			// Find local minimum in sm[] over the middle 60% of the run
			a := r[0] - yStart + (r[1]-r[0])*2/10
			b := r[0] - yStart + (r[1]-r[0])*8/10
			minI := a
			minV := sm[a]
			for i := a + 1; i <= b; i++ {
				if sm[i] < minV {
					minV = sm[i]
					minI = i
				}
			}
			cut := yStart + minI
			if cut-r[0] >= 8 && r[1]-cut >= 8 {
				split = append(split, [2]int{r[0], cut}, [2]int{cut, r[1]})
			} else {
				split = append(split, r)
			}
		}
		runs = split
	}
	return runs
}

func ocrCropField(img image.Image, x1, y1, x2, y2 int, fieldKind mgFieldKind, debugLabel string) string {
	bounds := img.Bounds()
	// Clamp
	if x1 < bounds.Min.X {
		x1 = bounds.Min.X
	}
	if y1 < bounds.Min.Y {
		y1 = bounds.Min.Y
	}
	if x2 > bounds.Max.X {
		x2 = bounds.Max.X
	}
	if y2 > bounds.Max.Y {
		y2 = bounds.Max.Y
	}
	cropW := x2 - x1
	cropH := y2 - y1
	if cropW <= 0 || cropH <= 0 {
		return ""
	}

	// Crop into RGBA buffer
	cropped := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	draw.Draw(cropped, cropped.Bounds(), img, image.Point{x1, y1}, draw.Src)

	// Preprocess (grayscale → invert → Otsu → pad → upscale)
	processed := preprocessForOCR(cropped)

	// Debug export (if MG_DEBUG_DIR set)
	if mgDebug != nil && debugLabel != "" {
		var col color.RGBA
		switch fieldKind {
		case mgFieldName:
			col = color.RGBA{0, 200, 0, 255} // green
		case mgFieldDamage:
			col = color.RGBA{255, 50, 50, 255} // red
		case mgFieldDate:
			col = color.RGBA{50, 100, 255, 255} // blue
		}
		mgDebugSaveCrop(debugLabel, cropped, processed, x1, y1, x2, y2, col)
	}

	// Encode for Tesseract
	var buf bytes.Buffer
	if err := png.Encode(&buf, processed); err != nil {
		return ""
	}

	// Configure Tesseract per field
	client := gosseract.NewClient()
	defer client.Close()
	client.SetImageFromBytes(buf.Bytes())

	switch fieldKind {
	case mgFieldDamage:
		// Damage value: digits, decimal punctuation variants, unit letters
		// Whitelist eliminates letter→digit confusion from label bleed.
		client.SetVariable("tessedit_char_whitelist", "0123456789.,GMKB")
		client.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
	case mgFieldName:
		// Name + alliance tag: letters, digits, brackets, space, underscore
		client.SetVariable("tessedit_char_whitelist",
			"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789[]_ ")
		client.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
	case mgFieldDate:
		client.SetVariable("tessedit_char_whitelist", "0123456789-: ")
		client.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
	}

	text, err := client.Text()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

// parseMGMaskName extracts player name and alliance tag from a clean OCR line.
// Input comes from whitelisted OCR (letters + digits + brackets + space + _),
// so the line is one of:
//   - "[RSRP]Gargolar"
//   - "[RSRP] Gargolar"    (space variant)
//   - "RSRP]Gargolar"      (missing opening bracket)
//   - "Gargolar"           (no tag)
//
// Brackets in the game font occasionally OCR as adjacent letters; we accept
// "I", "l", "1" as alternates only for the closing bracket where this is common.
func parseMGMaskName(text string) (name, tag string) {
	if text == "" {
		return "", ""
	}
	// Strip newlines (PSM_SINGLE_LINE should already give a single line, but be safe)
	line := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if line == "" {
		return "", ""
	}

	cleanName := func(s string) string {
		s = strings.TrimSpace(s)
		// Trim leading/trailing non-alphanumeric garbage
		s = regexp.MustCompile(`^[^A-Za-z0-9]+`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`[^A-Za-z0-9_ ]+$`).ReplaceAllString(s, "")
		// Collapse runs of spaces
		s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
		return strings.TrimSpace(s)
	}

	// Try [TAG]Name pattern
	if m := regexp.MustCompile(`\[([A-Z0-9]{2,6})[\]lI1]\s*(.+)`).FindStringSubmatch(line); m != nil {
		n := cleanName(m[2])
		if len(n) >= 2 {
			return n, m[1]
		}
	}
	// Try without opening bracket: "TAG]Name"
	if m := regexp.MustCompile(`([A-Z]{2,6})[\]lI1]\s*(.+)`).FindStringSubmatch(line); m != nil {
		n := cleanName(m[2])
		if len(n) >= 2 {
			return n, m[1]
		}
	}
	// No tag — return the cleaned line if it looks like a name
	cleaned := cleanName(line)
	if len(cleaned) >= 3 {
		return cleaned, ""
	}
	return "", ""
}

// parseMGMaskDamage extracts a damage value from a clean OCR line.
// Input comes from whitelisted OCR (digits + .,GMKB only), so we expect:
//   - "1.98G"  (clean)
//   - "835.58M"
//   - "1,98G"  (locale comma)
//   - "1.98"   (unit dropped)
//
// No more "Total[Damage:" garble — that label was cropped out before OCR.
func parseMGMaskDamage(text string) int64 {
	if text == "" {
		return 0
	}
	// Normalize: comma → dot, drop spaces, drop stray B (sometimes leaks from "Battle")
	t := strings.ReplaceAll(text, ",", ".")
	t = strings.ReplaceAll(t, " ", "")
	t = strings.ReplaceAll(t, "B", "")

	// Primary pattern: X.XX[GMK]
	if m := regexp.MustCompile(`(\d{1,3})\.(\d{1,2})([GMK])`).FindStringSubmatch(t); m != nil {
		intPart, _ := strconv.ParseFloat(m[1], 64)
		fracPart, _ := strconv.ParseFloat("0."+m[2], 64)
		val := intPart + fracPart
		return parseDamageValue(fmt.Sprintf("%.2f", val), m[3])
	}
	// Whole number with unit: "835M" — but check if decimal was dropped
	if m := regexp.MustCompile(`(\d+)([GMK])`).FindStringSubmatch(t); m != nil {
		num, _ := strconv.Atoi(m[1])
		unit := m[2]
		// If the number seems too large for the unit, a decimal point was likely missed.
		// Game values: G=1-99.99, M=1-999.99, K=1-999.99
		if (unit == "G" && num > 99) || (unit == "M" && num > 999) || (unit == "K" && num > 999) {
			// Game always shows exactly 2 decimal places (X.XX). Find split with 2-digit frac.
			numStr := m[1]
			for intLen := 1; intLen <= len(numStr)-2; intLen++ {
				if len(numStr)-intLen == 2 { // exactly 2 fraction digits
					intVal, _ := strconv.Atoi(numStr[:intLen])
					if (unit == "G" && intVal <= 99) || (unit == "M" && intVal <= 999) || (unit == "K" && intVal <= 999) {
						val, _ := strconv.ParseFloat(numStr[:intLen]+"."+numStr[intLen:], 64)
						return parseDamageValue(fmt.Sprintf("%.6f", val), unit)
					}
				}
			}
			// Fallback: try any valid split
			for intLen := 1; intLen <= 3 && intLen < len(numStr); intLen++ {
				intVal, _ := strconv.Atoi(numStr[:intLen])
				if (unit == "G" && intVal <= 99) || (unit == "M" && intVal <= 999) || (unit == "K" && intVal <= 999) {
					val, _ := strconv.ParseFloat(numStr[:intLen]+"."+numStr[intLen:], 64)
					return parseDamageValue(fmt.Sprintf("%.6f", val), unit)
				}
			}
		}
		return parseDamageValue(m[1], unit)
	}
	// Digits only (no decimal, no unit) — try to reconstruct.
	// "G" often misread as "6" at end, "0" can also be misread G.
	if m := regexp.MustCompile(`^(\d+)$`).FindStringSubmatch(t); m != nil {
		numStr := m[1]
		if len(numStr) >= 3 {
			lastCh := numStr[len(numStr)-1]
			var unit string
			switch lastCh {
			case '6': // G misread as 6
				unit = "G"
			case '0': // G misread as 0
				unit = "G"
			}
			if unit != "" {
				numStr = numStr[:len(numStr)-1]
				// Prefer split with 2-digit fraction (game format X.XX)
				for intLen := 1; intLen <= len(numStr)-2; intLen++ {
					if len(numStr)-intLen == 2 {
						intVal, _ := strconv.Atoi(numStr[:intLen])
						if (unit == "G" && intVal <= 99) || (unit == "M" && intVal <= 999) {
							val, _ := strconv.ParseFloat(numStr[:intLen]+"."+numStr[intLen:], 64)
							return parseDamageValue(fmt.Sprintf("%.6f", val), unit)
						}
					}
				}
				// Fallback: any valid split
				for intLen := 1; intLen <= 3 && intLen < len(numStr); intLen++ {
					intVal, _ := strconv.Atoi(numStr[:intLen])
					if (unit == "G" && intVal <= 99) || (unit == "M" && intVal <= 999) {
						val, _ := strconv.ParseFloat(numStr[:intLen]+"."+numStr[intLen:], 64)
						return parseDamageValue(fmt.Sprintf("%.6f", val), unit)
					}
				}
				// If no decimal needed
				return parseDamageValue(numStr, unit)
			}
		}
	}
	// Number with decimal but no unit — assume the unit was dropped; infer from magnitude
	if m := regexp.MustCompile(`(\d{1,3})\.(\d{1,2})`).FindStringSubmatch(t); m != nil {
		intPart, _ := strconv.ParseFloat(m[1], 64)
		fracPart, _ := strconv.ParseFloat("0."+m[2], 64)
		val := intPart + fracPart
		// Heuristic: G values are typically 1-99, M values are typically 1-999.
		// If int part ≤ 99, more likely G; otherwise M. (Sanity will check.)
		unit := "G"
		if intPart >= 100 {
			unit = "M"
		}
		return parseDamageValue(fmt.Sprintf("%.2f", val), unit)
	}
	return 0
}

// mgEnforceMonotonicity fixes damage values that violate descending rank order.
// In the game, player rows are always ranked highest-to-lowest damage.
// A leading "1" garble from OCR (e.g., "11.98G" instead of "1.98G") causes violations.
// Fix by stripping the leading digit from the integer part of the damage value.
func mgEnforceMonotonicity(participants []MGOCRParticipant) {
	for i := 1; i < len(participants); i++ {
		if participants[i].Damage <= 0 || participants[i-1].Damage <= 0 {
			continue
		}
		if participants[i].Damage > participants[i-1].Damage {
			// Violation: try stripping leading digit from the value
			fixed := mgStripLeadingDigit(participants[i].Damage)
			if fixed > 0 && fixed <= participants[i-1].Damage {
				log.Printf("MG mask monotonicity: row %d %q damage %d → %d",
					i, participants[i].NameSnapshot, participants[i].Damage, fixed)
				participants[i].Damage = fixed
			}
		}
	}
}

// mgStripLeadingDigit removes the leading digit from a damage value's integer part.
// E.g., 11.98G → 1.98G (removes leading "1"), 134.97M → 34.97M
func mgStripLeadingDigit(damage int64) int64 {
	// Determine unit magnitude
	var unit float64
	if damage >= 1000000000 {
		unit = 1000000000 // G
	} else if damage >= 1000000 {
		unit = 1000000 // M
	} else {
		return damage / 10
	}

	// Convert to float in the appropriate unit
	floatVal := float64(damage) / unit
	intPart := int(floatVal)
	fracPart := floatVal - float64(intPart)

	// Strip leading digit
	intStr := strconv.Itoa(intPart)
	if len(intStr) <= 1 {
		return 0 // can't strip from single digit
	}
	newInt, _ := strconv.Atoi(intStr[1:])
	newFloat := float64(newInt) + fracPart
	return int64(newFloat * unit)
}

// ocrMGAllVariants applies multiple image preprocessing strategies before OCR
// and returns ALL resulting texts (not just the best one) so the caller can merge.
func ocrMGAllVariants(imageData []byte) []string {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil
	}

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	// Minimal crop — skip top 5% (title bar) and bottom 3%
	cropTop := imgH * 5 / 100
	cropBottom := imgH * 97 / 100
	cropW := imgW
	cropH := cropBottom - cropTop

	cropped := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	draw.Draw(cropped, cropped.Bounds(), img, image.Point{bounds.Min.X, cropTop + bounds.Min.Y}, draw.Src)

	// Convert to grayscale
	gray := image.NewGray(image.Rect(0, 0, cropW, cropH))
	for y := 0; y < cropH; y++ {
		for x := 0; x < cropW; x++ {
			r, g, b, _ := cropped.At(x, y).RGBA()
			lum := uint8((299*r + 587*g + 114*b) / 1000 >> 8)
			gray.SetGray(x, y, color.Gray{Y: lum})
		}
	}

	// Strategy 1: Simple inversion (no threshold) — let Tesseract binarize
	inverted := image.NewGray(image.Rect(0, 0, cropW, cropH))
	for y := 0; y < cropH; y++ {
		for x := 0; x < cropW; x++ {
			px := gray.GrayAt(x, y).Y
			inverted.SetGray(x, y, color.Gray{Y: 255 - px})
		}
	}

	// Strategy 2: RED channel isolation + inversion
	// Game text (gold/white) has high Red; dark blue/purple BG has low Red
	redChan := image.NewGray(image.Rect(0, 0, cropW, cropH))
	for y := 0; y < cropH; y++ {
		for x := 0; x < cropW; x++ {
			r, _, _, _ := cropped.At(x, y).RGBA()
			redVal := uint8(r >> 8)
			// Invert: high red (text) → dark, low red (bg) → light
			redChan.SetGray(x, y, color.Gray{Y: 255 - redVal})
		}
	}

	// Strategy 3: Low threshold on grayscale (captures dim text)
	lowThreshold := uint8(40)
	binarizedLow := image.NewGray(image.Rect(0, 0, cropW, cropH))
	for y := 0; y < cropH; y++ {
		for x := 0; x < cropW; x++ {
			px := gray.GrayAt(x, y).Y
			if px > lowThreshold {
				binarizedLow.SetGray(x, y, color.Gray{Y: 0}) // text → black
			} else {
				binarizedLow.SetGray(x, y, color.Gray{Y: 255}) // bg → white
			}
		}
	}

	// Scale up all variants 3x (bigger = better for small damage text)
	scaledW := cropW * 3
	scaledH := cropH * 3

	scaledInv := image.NewGray(image.Rect(0, 0, scaledW, scaledH))
	scaledRed := image.NewGray(image.Rect(0, 0, scaledW, scaledH))
	scaledLow := image.NewGray(image.Rect(0, 0, scaledW, scaledH))
	for y := 0; y < scaledH; y++ {
		for x := 0; x < scaledW; x++ {
			scaledInv.SetGray(x, y, inverted.GrayAt(x/3, y/3))
			scaledRed.SetGray(x, y, redChan.GrayAt(x/3, y/3))
			scaledLow.SetGray(x, y, binarizedLow.GrayAt(x/3, y/3))
		}
	}

	// Run OCR on all variants, collect all texts
	variants := []image.Image{scaledInv, scaledRed, scaledLow}
	var allTexts []string

	for i, variant := range variants {
		var buf bytes.Buffer
		if err := png.Encode(&buf, variant); err != nil {
			continue
		}
		client := gosseract.NewClient()
		client.SetImageFromBytes(buf.Bytes())
		client.SetPageSegMode(gosseract.PSM_SINGLE_BLOCK)
		text, err := client.Text()
		client.Close()
		if err != nil {
			continue
		}
		log.Printf("MG OCR variant %d: %d lines", i, len(strings.Split(text, "\n")))
		allTexts = append(allTexts, text)
	}

	// Extra pass: red channel with PSM_SPARSE_TEXT to find scattered damage text
	{
		var buf bytes.Buffer
		if err := png.Encode(&buf, scaledRed); err == nil {
			client := gosseract.NewClient()
			client.SetImageFromBytes(buf.Bytes())
			client.SetPageSegMode(gosseract.PSM_SPARSE_TEXT)
			text, err := client.Text()
			client.Close()
			if err == nil && text != "" {
				log.Printf("MG OCR variant sparse: %d lines", len(strings.Split(text, "\n")))
				allTexts = append(allTexts, text)
			}
		}
	}

	return allTexts
}

// mergeOCRResults merges multiple OCR parse results into one.
// Uses the result with the most participants as the base, then fills in
// missing damage from other results using fuzzy name matching.
func mergeOCRResults(results []MarshalGuardOCRResult) MarshalGuardOCRResult {
	if len(results) == 0 {
		return MarshalGuardOCRResult{}
	}

	// Find the result with the most participants as our base
	bestIdx := 0
	bestCount := 0
	for i, r := range results {
		if len(r.Participants) > bestCount {
			bestCount = len(r.Participants)
			bestIdx = i
		}
	}

	merged := results[bestIdx]

	// Get event date and total damage from any result
	for _, r := range results {
		if r.EventDate != "" && merged.EventDate == "" {
			merged.EventDate = r.EventDate
		}
		if r.TotalDamage > merged.TotalDamage {
			merged.TotalDamage = r.TotalDamage
		}
	}

	// For each participant with missing damage, try to find it in other results
	for i, p := range merged.Participants {
		if p.Damage > 0 {
			continue
		}
		for ri, r := range results {
			if ri == bestIdx {
				continue
			}
			for _, sp := range r.Participants {
				if sp.Damage == 0 {
					continue
				}
				sim := calculateSimilarity(p.NameSnapshot, sp.NameSnapshot)
				if sim >= 70 || strings.EqualFold(p.NameSnapshot, sp.NameSnapshot) {
					merged.Participants[i].Damage = sp.Damage
					break
				}
			}
			if merged.Participants[i].Damage > 0 {
				break
			}
		}
	}

	// Monotonicity enforcement: damage must decrease with rank.
	// rank 1 (MVP) >= rank 2 >= rank 3 >= ... >= rank N
	// If a value breaks this, try dividing by 10 repeatedly to fit.
	// If it can't fit, discard it (set to 0 / N/A).
	for i := 1; i < len(merged.Participants); i++ {
		if merged.Participants[i].Damage == 0 {
			continue
		}
		// Find the nearest ranked-above participant with damage
		aboveDmg := int64(0)
		for j := i - 1; j >= 0; j-- {
			if merged.Participants[j].Damage > 0 {
				aboveDmg = merged.Participants[j].Damage
				break
			}
		}
		// Find the nearest ranked-below participant with damage
		belowDmg := int64(0)
		for j := i + 1; j < len(merged.Participants); j++ {
			if merged.Participants[j].Damage > 0 {
				belowDmg = merged.Participants[j].Damage
				break
			}
		}

		dmg := merged.Participants[i].Damage

		// Check: damage should be <= above (if above exists)
		if aboveDmg > 0 && dmg > aboveDmg {
			// Try dividing by powers of 10 to fit
			fixed := dmg
			for fixed > aboveDmg {
				fixed /= 10
			}
			if fixed >= belowDmg || belowDmg == 0 {
				log.Printf("MG OCR monotonicity fix: %q rank %d damage %d → %d (above=%d)",
					merged.Participants[i].NameSnapshot, i+1, dmg, fixed, aboveDmg)
				merged.Participants[i].Damage = fixed
			} else {
				log.Printf("MG OCR monotonicity discard: %q rank %d damage %d (above=%d below=%d)",
					merged.Participants[i].NameSnapshot, i+1, dmg, aboveDmg, belowDmg)
				merged.Participants[i].Damage = 0
			}
		}

		// Check: damage should be >= below (if below exists)
		if belowDmg > 0 && merged.Participants[i].Damage > 0 && merged.Participants[i].Damage < belowDmg {
			// Try multiplying by 10 to fit
			fixed := merged.Participants[i].Damage
			for fixed < belowDmg {
				fixed *= 10
			}
			if aboveDmg == 0 || fixed <= aboveDmg {
				log.Printf("MG OCR monotonicity fix up: %q rank %d damage %d → %d (below=%d)",
					merged.Participants[i].NameSnapshot, i+1, merged.Participants[i].Damage, fixed, belowDmg)
				merged.Participants[i].Damage = fixed
			}
		}
	}

	// Re-assign sequential ranks
	for i := range merged.Participants {
		merged.Participants[i].RankInEvent = i + 1
	}
	return merged
}

// computeOtsuThreshold calculates the optimal binarization threshold using Otsu's method.
func computeOtsuThreshold(gray *image.Gray) uint8 {
	bounds := gray.Bounds()
	histogram := make([]int, 256)
	total := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			histogram[gray.GrayAt(x, y).Y]++
			total++
		}
	}

	sum := 0
	for i := 0; i < 256; i++ {
		sum += i * histogram[i]
	}

	sumB := 0
	wB := 0
	maxVariance := 0.0
	threshold := uint8(128)

	for t := 0; t < 256; t++ {
		wB += histogram[t]
		if wB == 0 {
			continue
		}
		wF := total - wB
		if wF == 0 {
			break
		}
		sumB += t * histogram[t]
		mB := float64(sumB) / float64(wB)
		mF := float64(sum-sumB) / float64(wF)
		variance := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)
		if variance > maxVariance {
			maxVariance = variance
			threshold = uint8(t)
		}
	}
	return threshold
}

// parseMarshalGuardText extracts event data from OCR text of a Marshal Guard screenshot.
// The Alliance Exercise mail format has:
//   - [TAG]PlayerName on one line (relatively clean in OCR)
//   - "Total Damage: X.XXG" on the next line (often garbled by OCR)
//   - MVP section at the top (rank 1)
//   - Date at the bottom: "2026-5-6 20:30:10"
//
// Strategy: Find [TAG]Name patterns first (they survive OCR well as they are
// large bold text), then look for damage values in nearby lines using very
// flexible matching that handles OCR noise like: colons instead of dots,
// exclamation marks, brackets, missing characters, etc.
func parseMarshalGuardText(text string) MarshalGuardOCRResult {
	var result MarshalGuardOCRResult
	lines := strings.Split(text, "\n")

	log.Printf("MG OCR raw text (%d lines):\n%s", len(lines), text)

	// Extract event date — "2026-5-6 20:30:10" or similar
	dateRe := regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})`)
	for _, line := range lines {
		if m := dateRe.FindStringSubmatch(line); m != nil {
			year, _ := strconv.Atoi(m[1])
			month, _ := strconv.Atoi(m[2])
			day, _ := strconv.Atoi(m[3])
			if year >= 2024 && year <= 2030 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
				result.EventDate = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
				break
			}
		}
	}

	// ── Find [TAG]Name anchors ─────────────────────────────────────────────
	// Pattern for [TAG]Name — OCR often garbles the closing ] as l, I, or 1
	// Using uppercase-only for tag so backtracking works when ] is garbled
	tagNameRe := regexp.MustCompile(`\[([A-Z0-9]{2,6})[\]lI1)\s]([A-Za-z][A-Za-z0-9_ ]{1,25})`)

	// Flexible damage patterns that handle OCR noise:
	//   "27:35G" "27.35G" "27!35G" "27,35G" → 27.35G
	//   "12/08G" → 12.08G (slash as separator)
	//   "33'97M" → 33.97M (apostrophe as separator)
	//   "5:24G" "/5:24G" → 5.24G
	//   "643.05M" "643:05M" → 643.05M
	// Pattern: digits, OCR-separator (.:!,;|/'), two digits, optional unit
	dmgFlexRe := regexp.MustCompile(`(\d{1,4})[.:!,;|/''](\d{2})\s*([GMKgmk])?`)
	// Clean pattern: standard X.XXG
	dmgCleanRe := regexp.MustCompile(`(\d+\.?\d*)\s*([GMKgmk])`)
	// Bare number on damage line: OCR dropped decimal and/or unit (e.g., "1230" = 1.23G, "198" = 1.98G)
	dmgBareRe := regexp.MustCompile(`(\d{3,5})`)
	// OCR-garbled unit: digit(s) followed by char that could be G/M garbled (0, 6, 9, etc.)
	dmgGarbledUnitRe := regexp.MustCompile(`(\d{2,4})\s*[069O]`)

	// MVP marker
	mvpRe := regexp.MustCompile(`(?i)\bMV[PE]?\b`)
	// Attack count
	attackRe := regexp.MustCompile(`(?i)[Aa]ttacks?[:\s}]*(\d+)`)
	// "Damage Ranking" header — marks end of MVP section
	rankingHeaderRe := regexp.MustCompile(`(?i)[Dd]amage\s*[Rr]anking`)

	type entry struct {
		name    string
		tag     string
		damage  int64
		attacks *int
		isMVP   bool
	}
	var entries []entry

	// Find the "Damage Ranking" header line to separate MVP from ranked players
	rankingHeaderIdx := -1
	for i, line := range lines {
		if rankingHeaderRe.MatchString(line) {
			rankingHeaderIdx = i
			break
		}
	}

	// Check if MVP marker exists in the text
	hasMVP := false
	for _, line := range lines {
		if mvpRe.MatchString(line) {
			hasMVP = true
			break
		}
	}

	// Find all [TAG]Name occurrences
	usedDmgLines := map[int]bool{} // track which lines have been claimed for damage
	for i, line := range lines {
		m := tagNameRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		tag := m[1]
		name := strings.TrimSpace(m[2])

		// Skip UI labels
		lower := strings.ToLower(name)
		if strings.Contains(lower, "alliance") || strings.Contains(lower, "exercise") ||
			strings.Contains(lower, "reward") {
			continue
		}

		// Determine if this is the MVP (appears before the "Damage Ranking" header)
		isMVP := false
		if hasMVP && rankingHeaderIdx > 0 && i < rankingHeaderIdx {
			isMVP = true
		}

		// Look for damage value: search lines between this name and the next name.
		// Non-MVP: stop at the next [TAG]Name line. MVP has extra lines.
		var damage int64
		var attacks *int
		searchStart := i
		// Find where the next [TAG]Name starts (our search must stop before it)
		searchEnd := len(lines) - 1
		for nextI := i + 1; nextI < len(lines); nextI++ {
			if tagNameRe.MatchString(lines[nextI]) {
				searchEnd = nextI - 1
				break
			}
		}
		// Cap non-MVP to at most 2 lines forward, MVP up to 4
		maxEnd := i + 2
		if isMVP {
			maxEnd = i + 4
		}
		if searchEnd > maxEnd {
			searchEnd = maxEnd
		}
		if searchEnd >= len(lines) {
			searchEnd = len(lines) - 1
		}

		for si := searchStart; si <= searchEnd; si++ {
			sline := lines[si]

			// Skip lines that look like dates (e.g. "2026-5-6 20:30:10")
			if dateRe.MatchString(sline) {
				continue
			}

			// Skip lines already claimed for damage by a previous name
			if usedDmgLines[si] {
				continue
			}

			// Only look for damage on lines containing "amage" (from "Damage")
			// This prevents false matches on attack counts and other numbers
			amageIdx := strings.Index(strings.ToLower(sline), "amage")
			if amageIdx < 0 {
				// Still check for attack count on non-damage lines
				if am := attackRe.FindStringSubmatch(sline); am != nil {
					ac, _ := strconv.Atoi(am[1])
					attacks = &ac
				}
				continue
			}

			// Only search for damage numbers AFTER the "amage" keyword
			dmgPart := sline[amageIdx:]

			// Try flexible damage pattern (handles OCR noise)
			if dm := dmgFlexRe.FindStringSubmatch(dmgPart); dm != nil {
				intPart, _ := strconv.ParseFloat(dm[1], 64)
				fracPart, _ := strconv.ParseFloat("0."+dm[2], 64)
				val := intPart + fracPart
				unit := "G" // default to G if unit missing
				if dm[3] != "" {
					unit = strings.ToUpper(dm[3])
				}
				d := parseDamageValue(fmt.Sprintf("%.2f", val), unit)
				if d > damage {
					damage = d
					usedDmgLines[si] = true
				}
			} else if dm := dmgCleanRe.FindStringSubmatch(dmgPart); dm != nil {
				d := parseDamageValue(strings.ReplaceAll(dm[1], ",", ""), strings.ToUpper(dm[2]))
				if d > damage {
					damage = d
					usedDmgLines[si] = true
				}
			} else if dm := dmgBareRe.FindStringSubmatch(dmgPart); dm != nil {
				// Bare number without separator/unit: treat as G value
				// "1230" → 1230G (sanity check will fix to 1.23G)
				// "198" → 198G (sanity check will fix to 1.98G)
				val, _ := strconv.ParseInt(dm[1], 10, 64)
				if val > 0 {
					d := val * 1000000000 // treat as G
					if d > damage {
						damage = d
						usedDmgLines[si] = true
					}
				}
			} else if dm := dmgGarbledUnitRe.FindStringSubmatch(dmgPart); dm != nil {
				// "1230" where trailing "0" might be garbled "G"
				// "1796" where trailing "6" might be garbled "G"
				val, _ := strconv.ParseInt(dm[1], 10, 64)
				if val > 0 {
					d := val * 1000000000 // treat as G
					if d > damage {
						damage = d
						usedDmgLines[si] = true
					}
				}
			}
		}

		log.Printf("MG OCR found: name=%q tag=%q damage=%d isMVP=%v", name, tag, damage, isMVP)
		entries = append(entries, entry{name: name, tag: tag, damage: damage, attacks: attacks, isMVP: isMVP})
	}

	// Sanity check: non-MVP players cannot have more damage than MVP
	// If a value exceeds MVP, try recovery: OCR often drops decimal points
	// "198G" is likely "1.98G", "1279G" is likely "1.279G"
	var mvpDamage int64
	for _, e := range entries {
		if e.isMVP && e.damage > mvpDamage {
			mvpDamage = e.damage
		}
	}
	if mvpDamage > 0 {
		for i := range entries {
			if !entries[i].isMVP && entries[i].damage > mvpDamage {
				// Try dividing by increasing powers of 10 to find a valid value
				recovered := int64(0)
				for divisor := int64(10); divisor <= 1000; divisor *= 10 {
					candidate := entries[i].damage / divisor
					if candidate > 0 && candidate <= mvpDamage {
						recovered = candidate
						break
					}
				}
				if recovered > 0 {
					log.Printf("MG OCR sanity: %q damage %d exceeds MVP %d, recovered to %d", entries[i].name, entries[i].damage, mvpDamage, recovered)
					entries[i].damage = recovered
				} else {
					log.Printf("MG OCR sanity: %q damage %d exceeds MVP %d, discarding", entries[i].name, entries[i].damage, mvpDamage)
					entries[i].damage = 0
				}
			}
		}
	}

	// Assign ranks: MVP=1, rest sequential starting from 2
	rank := 2
	for _, e := range entries {
		r := rank
		if e.isMVP {
			r = 1
		} else {
			rank++
		}
		result.Participants = append(result.Participants, MGOCRParticipant{
			RankInEvent:  r,
			NameSnapshot: e.name,
			AllianceTag:  e.tag,
			Damage:       e.damage,
			AttackCount:  e.attacks,
		})
		log.Printf("MG OCR parsed rank=%d name=%q tag=%q damage=%d", r, e.name, e.tag, e.damage)
	}

	return result
}

func parseDamageValue(numStr, unit string) int64 {
	val, _ := strconv.ParseFloat(numStr, 64)
	switch strings.ToUpper(unit) {
	case "G":
		return int64(val * 1e9)
	case "M":
		return int64(val * 1e6)
	case "K":
		return int64(val * 1e3)
	}
	return int64(val)
}

func main() {
	// Initialize session store first
	initSessionStore()

	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	router := mux.NewRouter()

	// Auth routes (public)
	router.HandleFunc("/api/login", login).Methods("POST")
	router.HandleFunc("/api/logout", logout).Methods("POST")
	router.HandleFunc("/api/check-auth", checkAuth).Methods("GET")
	router.HandleFunc("/api/change-password", authMiddleware(changePassword)).Methods("POST")
	router.HandleFunc("/api/members/{id}/create-user", authMiddleware(adminR5Middleware(createUserForMember))).Methods("POST")

	// Admin routes (admin only)
	router.HandleFunc("/api/admin/users", authMiddleware(adminMiddleware(getAdminUsers))).Methods("GET")
	router.HandleFunc("/api/admin/users", authMiddleware(adminMiddleware(createAdminUser))).Methods("POST")
	router.HandleFunc("/api/admin/users/{id}", authMiddleware(adminMiddleware(updateAdminUser))).Methods("PUT")
	router.HandleFunc("/api/admin/users/{id}", authMiddleware(adminMiddleware(deleteAdminUser))).Methods("DELETE")
	router.HandleFunc("/api/admin/users/{id}/reset-password", authMiddleware(adminMiddleware(resetUserPassword))).Methods("POST")
	router.HandleFunc("/api/admin/login-history", authMiddleware(adminMiddleware(getLoginHistory))).Methods("GET")

	// API routes (protected)
	router.HandleFunc("/api/members", authMiddleware(getMembers)).Methods("GET")
	router.HandleFunc("/api/members/stats", authMiddleware(getMemberStats)).Methods("GET")
	router.HandleFunc("/api/members/archived", authMiddleware(getArchivedMembers)).Methods("GET")
	router.HandleFunc("/api/members", authMiddleware(rankManagementMiddleware(createMember))).Methods("POST")
	router.HandleFunc("/api/members/{id}/restore", authMiddleware(rankManagementMiddleware(restoreMember))).Methods("POST")
	router.HandleFunc("/api/members/{id}/permanent", authMiddleware(rankManagementMiddleware(permanentlyDeleteMember))).Methods("DELETE")
	router.HandleFunc("/api/members/{id}", authMiddleware(rankManagementMiddleware(updateMember))).Methods("PUT")
	router.HandleFunc("/api/members/{id}", authMiddleware(rankManagementMiddleware(deleteMember))).Methods("DELETE")
	router.HandleFunc("/api/members/import", authMiddleware(rankManagementMiddleware(importCSV))).Methods("POST")
	router.HandleFunc("/api/members/import/confirm", authMiddleware(rankManagementMiddleware(confirmMemberUpdates))).Methods("POST")
	router.HandleFunc("/api/members/import-screenshot", authMiddleware(rankManagementMiddleware(importMemberScreenshot))).Methods("POST")
	router.HandleFunc("/api/members/auto-register", authMiddleware(rankManagementMiddleware(autoRegisterMembers))).Methods("POST")

	// Train schedule routes (protected)
	router.HandleFunc("/api/train-schedules", authMiddleware(getTrainSchedules)).Methods("GET")
	router.HandleFunc("/api/train-schedules/weekly-message", authMiddleware(generateWeeklyMessage)).Methods("GET")
	router.HandleFunc("/api/train-schedules/daily-message", authMiddleware(generateDailyMessage)).Methods("GET")
	router.HandleFunc("/api/train-schedules/conductor-messages", authMiddleware(generateConductorMessages)).Methods("GET")
	router.HandleFunc("/api/train-schedules/auto-schedule", authMiddleware(autoSchedule)).Methods("POST")
	router.HandleFunc("/api/train-schedules/lucky-draw", authMiddleware(rankManagementMiddleware(luckyDraw))).Methods("POST")
	router.HandleFunc("/api/train-schedules", authMiddleware(createTrainSchedule)).Methods("POST")
	router.HandleFunc("/api/train-schedules/{id}", authMiddleware(updateTrainSchedule)).Methods("PUT")
	router.HandleFunc("/api/train-schedules/{id}", authMiddleware(deleteTrainSchedule)).Methods("DELETE")

	// Awards routes (protected)
	router.HandleFunc("/api/awards", authMiddleware(getAwards)).Methods("GET")
	router.HandleFunc("/api/awards", authMiddleware(saveAwards)).Methods("POST")
	router.HandleFunc("/api/awards/{week}", authMiddleware(deleteWeekAwards)).Methods("DELETE")

	// Award types routes
	router.HandleFunc("/api/award-types", authMiddleware(getAwardTypes)).Methods("GET")
	router.HandleFunc("/api/award-types", authMiddleware(createAwardType)).Methods("POST")
	router.HandleFunc("/api/award-types/{id}", authMiddleware(updateAwardType)).Methods("PUT")
	router.HandleFunc("/api/award-types/{id}", authMiddleware(deleteAwardType)).Methods("DELETE")

	// VS points routes (protected)
	router.HandleFunc("/api/vs-points", authMiddleware(getVSPoints)).Methods("GET")
	router.HandleFunc("/api/vs-points", authMiddleware(saveVSPoints)).Methods("POST")
	router.HandleFunc("/api/vs-points/{week}", authMiddleware(deleteWeekVSPoints)).Methods("DELETE")
	router.HandleFunc("/api/vs-points/process-screenshot", authMiddleware(processVSPointsScreenshot)).Methods("POST")
	router.HandleFunc("/api/vs-points/patch", authMiddleware(r4r5Middleware(patchVSPoint))).Methods("PATCH")
	router.HandleFunc("/api/vs-compliance", authMiddleware(getVSCompliance)).Methods("GET")

	// Recommendations routes (protected)
	router.HandleFunc("/api/recommendations", authMiddleware(getRecommendations)).Methods("GET")
	router.HandleFunc("/api/recommendations", authMiddleware(createRecommendation)).Methods("POST")
	router.HandleFunc("/api/recommendations/{id}", authMiddleware(deleteRecommendation)).Methods("DELETE")

	// Conduct Reports routes (protected)
	router.HandleFunc("/api/conduct-reports", authMiddleware(getConductReports)).Methods("GET")
	router.HandleFunc("/api/conduct-reports", authMiddleware(createConductReport)).Methods("POST")
	router.HandleFunc("/api/conduct-reports/{id}", authMiddleware(deleteConductReport)).Methods("DELETE")

	// Settings routes (protected)
	router.HandleFunc("/api/settings", authMiddleware(getSettings)).Methods("GET")
	router.HandleFunc("/api/settings", authMiddleware(adminR5Middleware(updateSettings))).Methods("PUT")
	router.HandleFunc("/api/settings/backup-rotation", authMiddleware(getBackupRotation)).Methods("GET")
	router.HandleFunc("/api/settings/backup-rotation", authMiddleware(rankManagementMiddleware(saveBackupRotation))).Methods("PUT")
	router.HandleFunc("/api/settings/train-week-mode", authMiddleware(getTrainWeekMode)).Methods("GET")
	router.HandleFunc("/api/settings/train-week-mode", authMiddleware(rankManagementMiddleware(setTrainWeekMode))).Methods("PUT")

	// Applicant/Recruitment routes (protected)
	router.HandleFunc("/api/applicants", authMiddleware(getApplicants)).Methods("GET")
	router.HandleFunc("/api/applicants", authMiddleware(rankManagementMiddleware(createApplicant))).Methods("POST")
	router.HandleFunc("/api/applicants/{id}/status", authMiddleware(rankManagementMiddleware(updateApplicantStatus))).Methods("PUT")
	router.HandleFunc("/api/applicants/{id}", authMiddleware(rankManagementMiddleware(updateApplicant))).Methods("PUT")
	router.HandleFunc("/api/applicants/{id}", authMiddleware(adminR5Middleware(deleteApplicant))).Methods("DELETE")

	// Rankings routes (protected)
	router.HandleFunc("/api/rankings", authMiddleware(getMemberRankings)).Methods("GET")
	router.HandleFunc("/api/member-timelines", authMiddleware(getMemberTimelines)).Methods("GET")

	// Storm assignments routes (protected, R4/R5 only)
	router.HandleFunc("/api/storm-assignments", authMiddleware(getStormAssignments)).Methods("GET")
	router.HandleFunc("/api/storm-assignments", authMiddleware(r4r5Middleware(saveStormAssignments))).Methods("POST")
	router.HandleFunc("/api/storm-assignments/{taskForce}", authMiddleware(r4r5Middleware(deleteStormAssignments))).Methods("DELETE")

	// Power history routes (protected)
	router.HandleFunc("/api/power-history", authMiddleware(getPowerHistory)).Methods("GET")
	router.HandleFunc("/api/power-history", authMiddleware(addPowerRecord)).Methods("POST")
	router.HandleFunc("/api/power-history/process-screenshot", authMiddleware(processPowerScreenshot)).Methods("POST")

	// Marshal Guard routes (protected)
	router.HandleFunc("/api/marshal-guard", authMiddleware(listMarshalGuardEvents)).Methods("GET")
	router.HandleFunc("/api/marshal-guard", authMiddleware(r3PlusMiddleware(createMarshalGuardEvent))).Methods("POST")
	router.HandleFunc("/api/marshal-guard/process-screenshots", authMiddleware(r3PlusMiddleware(processMarshalGuardScreenshots))).Methods("POST")
	router.HandleFunc("/api/marshal-guard/process-mg-v2", authMiddleware(r3PlusMiddleware(processMGV2))).Methods("POST")
	router.HandleFunc("/api/marshal-guard/confirm", authMiddleware(r3PlusMiddleware(confirmMarshalGuard))).Methods("POST")
	router.HandleFunc("/api/marshal-guard/member-stats", authMiddleware(getMarshalGuardMemberStats)).Methods("GET")
	router.HandleFunc("/api/marshal-guard/{id}", authMiddleware(getMarshalGuardEvent)).Methods("GET")
	router.HandleFunc("/api/marshal-guard/{id}", authMiddleware(rankManagementMiddleware(updateMarshalGuardEvent))).Methods("PUT")
	router.HandleFunc("/api/marshal-guard/{id}", authMiddleware(rankManagementMiddleware(deleteMarshalGuardEvent))).Methods("DELETE")
	router.HandleFunc("/api/marshal-guard/{id}/participants/{pid}", authMiddleware(rankManagementMiddleware(updateMarshalGuardParticipant))).Methods("PUT")

	// Health check endpoints (no auth)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")
	router.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","error":"database unavailable"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods("GET")

	// Serve static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", requestLoggingMiddleware(router)))
}
