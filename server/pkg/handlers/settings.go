package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type SettingsHandler struct {
	users     *repository.UserRepository
	groups    *repository.GroupRepository
	budgets   *repository.BudgetRepository
	goals     *repository.GoalRepository
	db        *mongo.Database
	startedAt time.Time
	version   string
}

func NewSettingsHandler(
	users *repository.UserRepository,
	groups *repository.GroupRepository,
	budgets *repository.BudgetRepository,
	goals *repository.GoalRepository,
	db *mongo.Database,
	startedAt time.Time,
	version string,
) *SettingsHandler {
	return &SettingsHandler{
		users:     users,
		groups:    groups,
		budgets:   budgets,
		goals:     goals,
		db:        db,
		startedAt: startedAt,
		version:   version,
	}
}

// ServerInfo is the server identity block.
type ServerInfo struct {
	Host          string    `json:"host"`
	Online        bool      `json:"online"`
	StartedAt     time.Time `json:"started_at"`
	Uptime        string    `json:"uptime"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	Version       string    `json:"version"`
}

// ServerStats is the runtime/db stats block.
type ServerStats struct {
	Users       int64  `json:"users"`
	Groups      int64  `json:"groups"`
	Budgets     int64  `json:"budgets"`
	Goals       int64  `json:"goals"`
	DbSizeBytes int64  `json:"db_size_bytes"`
	MemMB       uint64 `json:"mem_mb"`
	GoRoutines  int    `json:"goroutines"`
}

// ProfileInfo is the logged-in user block.
type ProfileInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// SettingsResponse is the full payload of GET /settings.
type SettingsResponse struct {
	Server  ServerInfo  `json:"server"`
	Stats   ServerStats `json:"stats"`
	Profile ProfileInfo `json:"profile"`
}

// Get returns server identity, stats and the authenticated user's profile.
// @Summary Get settings snapshot
// @Description Returns server identity (host/uptime/version), runtime stats and the authenticated user's profile
// @Tags settings
// @Produce json
// @Success 200 {object} handlers.SettingsResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/settings [get]
func (h *SettingsHandler) Get(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	ctx := context.Background()

	uptime := time.Since(h.startedAt)

	var usersCount, groupsCount int64
	if users, err := h.users.List(ctx); err == nil {
		usersCount = int64(len(users))
	}
	if groups, err := h.groups.List(ctx); err == nil {
		groupsCount = int64(len(groups))
	}
	budgetsCount, _ := h.budgets.CountByUser(ctx, userID)

	var dbSizeBytes int64
	if h.db != nil {
		var stats struct {
			DataSize int64 `bson:"dataSize"`
			FSSize   int64 `bson:"fsUsedSize"`
		}
		_ = h.db.RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&stats)
		if stats.FSSize > 0 {
			dbSizeBytes = stats.FSSize
		} else {
			dbSizeBytes = stats.DataSize
		}
	}

	goalsAll, _ := h.goals.ListByUser(ctx, userID)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	resp := SettingsResponse{
		Server: ServerInfo{
			Host:          string(c.Request().URI().Host()),
			Online:        true,
			StartedAt:     h.startedAt,
			Uptime:        formatUptime(uptime),
			UptimeSeconds: int64(uptime.Seconds()),
			Version:       h.version,
		},
		Stats: ServerStats{
			Users:       usersCount,
			Groups:      groupsCount,
			Budgets:     budgetsCount,
			Goals:       int64(len(goalsAll)),
			DbSizeBytes: dbSizeBytes,
			MemMB:       m.Alloc / 1024 / 1024,
			GoRoutines:  runtime.NumGoroutine(),
		},
	}

	if u, err := h.users.GetByID(ctx, userID); err == nil && u != nil {
		resp.Profile = ProfileInfo{
			ID:    u.ID.Hex(),
			Email: u.Email,
			Name:  u.Name,
		}
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// formatUptime pretty-prints a duration as "12d 04h 17m" / "04h 17m 32s".
func formatUptime(d time.Duration) string {
	total := int64(d.Seconds())
	days := total / 86400
	hours := (total % 86400) / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	if days > 0 {
		return fmt.Sprintf("%dd %02dh %02dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%02dh %02dm %02ds", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02dm %02ds", minutes, seconds)
}
