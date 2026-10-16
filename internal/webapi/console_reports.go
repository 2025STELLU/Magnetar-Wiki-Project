package webapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2025STELLU/MagnetarWiki/internal/auth"
	"github.com/2025STELLU/MagnetarWiki/internal/csrf"
	"github.com/2025STELLU/MagnetarWiki/internal/db"
	"github.com/2025STELLU/MagnetarWiki/internal/i18n"
	"github.com/2025STELLU/MagnetarWiki/internal/perms"
	"github.com/2025STELLU/MagnetarWiki/internal/repo"
	"github.com/2025STELLU/MagnetarWiki/internal/site"
	"github.com/2025STELLU/MagnetarWiki/internal/wikijson"
)

// ReportsPrefix is the JSON API root for the console report queue.
const ReportsPrefix = "/pw-api/console/reports/"

// ConsoleReports serves the report-handling page of the new console.
// Wiring (mux registration) is done by the parent; this file only defines
// the handler.
type ConsoleReports struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleReports)(nil)

func NewConsoleReports(d Deps, next http.Handler) *ConsoleReports {
	return &ConsoleReports{deps: d, next: next}
}

func (h *ConsoleReports) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, ReportsPrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	head, _, _ := strings.Cut(rest, "/")
	switch {
	case head == "list" && r.Method == http.MethodGet:
		h.list(w, r, loc)
	case head == "detail" && r.Method == http.MethodGet:
		h.detail(w, r, loc)
	case head == "handle" && r.Method == http.MethodPost:
		h.handle(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// reportsAllowed mirrors Console.allowed: login + the named permission.
func (h *ConsoleReports) reportsAllowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return false
	}
	subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(user, time.Now())
	if err != nil {
		h.deps.log().Error("resolve permissions", "user", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return false
	}
	if !perms.Resolve(subject, nil).Has(perms.ViewUserReports) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

// apiStatus maps the console tab to the stored report status.
// "all" (and anything unknown) means no filter.
func apiStatus(tab string) string {
	switch tab {
	case "pending":
		return db.ReportPending
	case "handled":
		return db.ReportReviewed
	case "dismissed":
		return db.ReportDismissed
	default:
		return ""
	}
}

func reportListJSON(rows []db.ReportRow, total int) wikijson.Object {
	items := make(wikijson.Array, 0, len(rows))
	for _, r := range rows {
		items = append(items, wikijson.Object{
			{Key: "id", Value: r.ID},
			{Key: "reporter", Value: r.Reporter},
			{Key: "reported", Value: r.Reported},
			{Key: "reason", Value: r.Reason},
			{Key: "status", Value: r.Status},
			{Key: "created_at", Value: r.CreatedAt.Format(time.RFC3339)},
		})
	}
	return wikijson.Object{
		{Key: "reports", Value: items},
		{Key: "total", Value: total},
	}
}

func (h *ConsoleReports) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.reportsAllowed(w, r, loc) {
		return
	}
	q := r.URL.Query()
	status := apiStatus(q.Get("status"))
	limit := 20
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}
	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n >= 0 {
		offset = n
	}
	rows, total, err := h.deps.DB.AdminReports(r.Context(), siteID(r.Context()), status, limit, offset)
	if err != nil {
		h.deps.log().Error("list reports", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(reportListJSON(rows, total))
	if err != nil {
		h.deps.log().Error("marshal report list", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type reportMessage struct {
	SenderName string    `json:"sender_name"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// reportMessagesJSON parses the stored message snapshot for the frontend.
// Unparseable snapshots are passed through as a raw string.
func reportMessagesJSON(raw string) any {
	var found []reportMessage
	if err := json.Unmarshal([]byte(raw), &found); err != nil {
		return raw
	}
	out := make(wikijson.Array, 0, len(found))
	for _, m := range found {
		out = append(out, wikijson.Object{
			{Key: "sender_name", Value: m.SenderName},
			{Key: "body", Value: m.Body},
			{Key: "created_at", Value: m.CreatedAt.Format(time.RFC3339)},
		})
	}
	return out
}

func reportDetailJSON(r db.ReportRow) wikijson.Object {
	var reviewedAt any
	if r.ReviewedAt != nil {
		reviewedAt = r.ReviewedAt.Format(time.RFC3339)
	}
	return wikijson.Object{
		{Key: "id", Value: r.ID},
		{Key: "reporter", Value: r.Reporter},
		{Key: "reported", Value: r.Reported},
		{Key: "reason", Value: r.Reason},
		{Key: "messages", Value: reportMessagesJSON(r.Messages)},
		{Key: "status", Value: r.Status},
		{Key: "admin_notes", Value: r.AdminNotes},
		{Key: "reply", Value: r.Reply},
		{Key: "created_at", Value: r.CreatedAt.Format(time.RFC3339)},
		{Key: "reviewed_at", Value: reviewedAt},
		{Key: "reviewed_by", Value: r.ReviewedBy},
	}
}

func (h *ConsoleReports) detail(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.reportsAllowed(w, r, loc) {
		return
	}
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	row, err := h.deps.DB.AdminReport(r.Context(), siteID(r.Context()), id)
	if err != nil {
		h.deps.log().Error("read report", "id", id, "err", err)
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-page-not-found")))
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "report", Value: reportDetailJSON(row)}})
	if err != nil {
		h.deps.log().Error("marshal report detail", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type handleReportBody struct {
	ID     int64  `json:"id"`
	Action string `json:"action"`
	Note   string `json:"note"`
}

func (h *ConsoleReports) handle(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-page-not-found")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		refuseCSRF(w, r, loc)
		return
	}
	if !h.reportsAllowed(w, r, loc) {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var body handleReportBody
	if err := json.Unmarshal(raw, &body); err != nil || body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var status string
	switch body.Action {
	case "dismiss":
		status = db.ReportDismissed
	case "action_taken":
		status = db.ReportReviewed
	default:
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	mine := auth.FromContext(ctx)
	if mine == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	note := strings.TrimSpace(body.Note)
	if len(note) > 2000 {
		note = note[:2000]
	}
	if err := h.deps.DB.ReviewReport(ctx, siteID(ctx), body.ID, status, note, "", mine.ID, time.Now()); err != nil {
		h.deps.log().Error("handle report", "id", body.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	logAdminAction(ctx, h.deps, db.AdminChanged, "reports", body.ID, "")
	resp, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal handle result", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
