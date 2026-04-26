package handlers

import (
	"archive/zip"
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"runtime"
	"sort"
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
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	Name     string  `json:"name"`
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance"`
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
		cur := u.Currency
		if cur == "" {
			cur = "UZS"
		}
		resp.Profile = ProfileInfo{
			ID:       u.ID.Hex(),
			Email:    u.Email,
			Name:     u.Name,
			Currency: cur,
			Balance:  u.Balance,
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

func (h *SettingsHandler) exportDataAsJSON(c fiber.Ctx) error {
	// dump all collections from mongodatabase
	collections, err := h.db.ListCollectionNames(context.Background(), bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to list collections", nil))
	}

	all_data := make([]interface{}, 0)

	for _, collection := range collections {
		collectionData, err := h.db.Collection(collection).Find(context.Background(), bson.M{})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to dump collection data", nil))
		}
		data := make([]interface{}, 0)
		err = collectionData.All(context.Background(), &data)
		if err != nil {
			log.Printf("failed to dump collection data: %v", err)
		}
		all_data = append(all_data, data)
	}
	return c.Status(fiber.StatusOK).JSON(repository.NewResponse("success", "data exported successfully", all_data))

}

// exportDataAsCSV streams a ZIP archive containing one CSV per Mongo collection.
// Columns are the union of every document's top-level keys in that collection,
// sorted with `_id` first. Nested values (maps/arrays) are serialized as JSON strings.
func (h *SettingsHandler) exportDataAsCSV(c fiber.Ctx) error {
	ctx := context.Background()

	collections, err := h.db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to list collections", nil))
	}

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	for _, name := range collections {
		cur, err := h.db.Collection(name).Find(ctx, bson.M{})
		if err != nil {
			_ = zw.Close()
			return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to read collection "+name, nil))
		}
		var docs []bson.M
		if err := cur.All(ctx, &docs); err != nil {
			_ = zw.Close()
			return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to decode collection "+name, nil))
		}

		// Union of keys across all docs, ordered: _id first, then sorted.
		seen := map[string]struct{}{}
		for _, d := range docs {
			for k := range d {
				seen[k] = struct{}{}
			}
		}
		keys := make([]string, 0, len(seen))
		for k := range seen {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if _, ok := seen["_id"]; ok {
			// pull "_id" to the front
			out := []string{"_id"}
			for _, k := range keys {
				if k != "_id" {
					out = append(out, k)
				}
			}
			keys = out
		}

		w, err := zw.Create(name + ".csv")
		if err != nil {
			_ = zw.Close()
			return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to create csv entry", nil))
		}
		cw := csv.NewWriter(w)
		if len(keys) > 0 {
			if err := cw.Write(keys); err != nil {
				_ = zw.Close()
				return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to write csv header", nil))
			}
			for _, d := range docs {
				row := make([]string, len(keys))
				for i, k := range keys {
					row[i] = csvCell(d[k])
				}
				if err := cw.Write(row); err != nil {
					_ = zw.Close()
					return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to write csv row", nil))
				}
			}
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
			_ = zw.Close()
			return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "csv flush failed", nil))
		}
	}

	if err := zw.Close(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to finalize zip", nil))
	}

	filename := fmt.Sprintf("gift-export-%s.zip", time.Now().UTC().Format("20060102-150405"))
	c.Set(fiber.HeaderContentType, "application/zip")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Status(fiber.StatusOK).Send(buf.Bytes())
}

// csvCell renders any bson value as a single CSV cell. ObjectIDs → hex,
// times → RFC3339, primitives → Sprint, everything else → JSON blob.
func csvCell(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bson.ObjectID:
		return x.Hex()
	case time.Time:
		return x.UTC().Format(time.RFC3339)
	case bool, int, int32, int64, float32, float64:
		return fmt.Sprint(x)
	default:
		// bson.M / bson.A / slice — marshal as extended JSON for round-trippability
		if b, err := bson.MarshalExtJSON(v, false, false); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", v)
	}
}

// ExportData godoc
// @Summary      Export all data
// @Description  Export all collections data from the database in JSON or CSV format
// @Tags         settings
// @Produce      json
// @Param        format  query     string  false  "Export format: json or csv"  Enums(json, csv)  default(json)
// @Success      200     {object}  map[string]interface{}  "Success"
// @Failure      400     {object}  map[string]string       "Invalid format"
// @Failure      500     {object}  map[string]interface{}  "Internal server error"
// @Router       /api/v1/settings/export_data [post]
func (h *SettingsHandler) ExportData(c fiber.Ctx) error {
	export_to_format := c.Query("format", "json")

	switch export_to_format {
	case "json":
		return h.exportDataAsJSON(c)
	case "csv":
		return h.exportDataAsCSV(c)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid format"})
	}
}
