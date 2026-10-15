package webapi

import (
	"context"
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

// TicketsPrefix is the JSON API root for the console ticket queue
// (user tickets and membership applications).
const TicketsPrefix = "/pw-api/console/tickets/"

// ConsoleTickets serves the ticket/application handling page of the new console.
// Wiring (mux registration) is done by the parent; this file only defines
// the handler.
type ConsoleTickets struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleTickets)(nil)

func NewConsoleTickets(d Deps, next http.Handler) *ConsoleTickets {
	return &ConsoleTickets{deps: d, next: next}
}

func (h *ConsoleTickets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, TicketsPrefix)
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
	case head == "reply" && r.Method == http.MethodPost:
		h.reply(w, r, loc)
	case head == "handle" && r.Method == http.MethodPost:
		h.handle(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// ticketPerm picks the permission guarding a ticket kind, mirroring the old
// admin screens: tickets need ViewUserTickets, membership applications need
// ReviewMembershipApplications.
func ticketPerm(kind string) string {
	if kind == db.MembershipApplyKind {
		return perms.ReviewMembershipApplications
	}
	return perms.ViewUserTickets
}

// ticketsAllowed mirrors Console.allowed: login + the kind's permission.
func (h *ConsoleTickets) ticketsAllowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, kind string) bool {
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
	if !perms.Resolve(subject, nil).Has(ticketPerm(kind)) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

// apiTicketKind maps the console tab to the stored ticket kind.
func apiTicketKind(tab string) (string, bool) {
	switch tab {
	case "ticket":
		return db.TicketKind, true
	case "application":
		return db.MembershipApplyKind, true
	default:
		return "", false
	}
}

// apiTicketStatus maps the console tab to the stored ticket status.
// "all" (and anything unknown) means no filter.
func apiTicketStatus(tab string) string {
	switch tab {
	case db.TicketPending, db.TicketApproved, db.TicketRejected, db.TicketClosed:
		return tab
	default:
		return ""
	}
}

func ticketListJSON(rows []db.TicketRow, total int) wikijson.Object {
	items := make(wikijson.Array, 0, len(rows))
	for _, t := range rows {
		items = append(items, wikijson.Object{
			{Key: "id", Value: t.ID},
			{Key: "kind", Value: t.Kind},
			{Key: "author", Value: t.Author},
			{Key: "subject", Value: t.Subject},
			{Key: "status", Value: t.Status},
			{Key: "created_at", Value: t.CreatedAt.Format(time.RFC3339)},
		})
	}
	return wikijson.Object{
		{Key: "items", Value: items},
		{Key: "total", Value: total},
	}
}

func (h *ConsoleTickets) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	q := r.URL.Query()
	kind, ok := apiTicketKind(q.Get("type"))
	if !ok {
		// 鉴权先行：未登录一律 401，不因参数问题泄露接口存在性
		if auth.FromContext(r.Context()) == nil {
			writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
			return
		}
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if !h.ticketsAllowed(w, r, loc, kind) {
		return
	}
	status := apiTicketStatus(q.Get("status"))
	limit := 20
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}
	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n >= 0 {
		offset = n
	}
	rows, total, err := h.deps.DB.AdminTickets(r.Context(), siteID(r.Context()), kind, status, limit, offset)
	if err != nil {
		h.deps.log().Error("list tickets", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(ticketListJSON(rows, total))
	if err != nil {
		h.deps.log().Error("marshal ticket list", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// ticketMessagesJSON builds the conversation flow for the frontend from the
// stored body (submitter) and reply (admin). There is no separate message
// table; these two fields are the whole history.
func ticketMessagesJSON(t db.TicketRow) wikijson.Array {
	out := make(wikijson.Array, 0, 2)
	out = append(out, wikijson.Object{
		{Key: "sender_name", Value: t.Author},
		{Key: "body", Value: t.Body},
		{Key: "created_at", Value: t.CreatedAt.Format(time.RFC3339)},
	})
	if strings.TrimSpace(t.Reply) != "" {
		at := t.CreatedAt.Format(time.RFC3339)
		if t.ReviewedAt != nil {
			at = t.ReviewedAt.Format(time.RFC3339)
		}
		who := t.ReviewedBy
		if who == "" {
			who = "admin"
		}
		out = append(out, wikijson.Object{
			{Key: "sender_name", Value: who},
			{Key: "body", Value: t.Reply},
			{Key: "created_at", Value: at},
		})
	}
	return out
}

func ticketDetailJSON(t db.TicketRow) wikijson.Object {
	var reviewedAt any
	if t.ReviewedAt != nil {
		reviewedAt = t.ReviewedAt.Format(time.RFC3339)
	}
	var grantedID any
	if t.GrantedID != nil {
		grantedID = *t.GrantedID
	}
	return wikijson.Object{
		{Key: "id", Value: t.ID},
		{Key: "kind", Value: t.Kind},
		{Key: "author", Value: t.Author},
		{Key: "subject", Value: t.Subject},
		{Key: "body", Value: t.Body},
		{Key: "source_page", Value: t.SourcePage},
		{Key: "status", Value: t.Status},
		{Key: "admin_notes", Value: t.AdminNotes},
		{Key: "reply", Value: t.Reply},
		{Key: "messages", Value: ticketMessagesJSON(t)},
		{Key: "created_at", Value: t.CreatedAt.Format(time.RFC3339)},
		{Key: "reviewed_at", Value: reviewedAt},
		{Key: "reviewed_by", Value: t.ReviewedBy},
		{Key: "granted_role_id", Value: grantedID},
	}
}

func (h *ConsoleTickets) detail(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	row, err := h.deps.DB.AdminTicket(r.Context(), siteID(r.Context()), id)
	if err != nil {
		h.deps.log().Error("read ticket", "id", id, "err", err)
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-page-not-found")))
		return
	}
	if !h.ticketsAllowed(w, r, loc, row.Kind) {
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "ticket", Value: ticketDetailJSON(row)}})
	if err != nil {
		h.deps.log().Error("marshal ticket detail", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *ConsoleTickets) checkCSRF(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
	current := site.FromContext(r.Context())
	if current == nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-page-not-found")))
		return false
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		refuseCSRF(w, r, loc)
		return false
	}
	return true
}

func readTicketBody(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, v any) bool {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return false
	}
	if err := json.Unmarshal(raw, v); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return false
	}
	return true
}

// ticketWorthTelling mirrors the old admin's worthTelling: only ping the
// submitter when something actually changed and it wasn't their own action.
func ticketWorthTelling(to *int64, by int64, beforeStatus, beforeReply, afterStatus, afterReply string) bool {
	if to == nil || *to == by || afterStatus == db.TicketPending {
		return false
	}
	return afterStatus != beforeStatus || afterReply != beforeReply
}

func (h *ConsoleTickets) tellSubmitter(ctx context.Context, row db.TicketRow, by int64, status, reply string) {
	if !ticketWorthTelling(row.AuthorID, by, row.Status, row.Reply, status, reply) {
		return
	}
	meta, err := wikijson.Marshal(wikijson.Object{
		{Key: "kind", Value: row.Kind},
		{Key: "ticket_id", Value: row.ID},
		{Key: "subject", Value: row.Subject},
		{Key: "status", Value: status},
		{Key: "reply", Value: reply},
	})
	if err == nil {
		err = h.deps.DB.SendNotification(ctx, siteID(ctx), db.NotifyTicketResult, meta, []int64{*row.AuthorID}, time.Now())
	}
	if err != nil {
		h.deps.log().Error("tell the submitter", "kind", row.Kind, "id", row.ID, "err", err)
	}
}

type replyTicketBody struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

// reply records an admin reply without changing the ticket status.
func (h *ConsoleTickets) reply(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	ctx := r.Context()
	if !h.checkCSRF(w, r, loc) {
		return
	}
	var body replyTicketBody
	if !readTicketBody(w, r, loc, &body) || body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	row, err := h.deps.DB.AdminTicket(ctx, siteID(ctx), body.ID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-page-not-found")))
		return
	}
	if !h.ticketsAllowed(w, r, loc, row.Kind) {
		return
	}
	message := strings.TrimSpace(body.Message)
	if message == "" || len(message) > 2000 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	mine := auth.FromContext(ctx)
	if mine == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	// Keep status/notes/grant as they are; only the reply changes.
	if err := h.deps.DB.ReviewTicket(ctx, siteID(ctx), row.ID, row.Status, row.AdminNotes, message, mine.ID, row.GrantedID, time.Now()); err != nil {
		h.deps.log().Error("reply ticket", "id", row.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	h.tellSubmitter(ctx, row, mine.ID, row.Status, message)
	writeOK(w, loc)
}

type handleTicketBody struct {
	ID           int64  `json:"id"`
	Action       string `json:"action"`
	Note         string `json:"note"`
	GrantedRole  *int64 `json:"granted_role_id"`
}

// handle moves a ticket/application to approved, rejected or closed.
func (h *ConsoleTickets) handle(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	ctx := r.Context()
	if !h.checkCSRF(w, r, loc) {
		return
	}
	var body handleTicketBody
	if !readTicketBody(w, r, loc, &body) || body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var status string
	switch body.Action {
	case "approve":
		status = db.TicketApproved
	case "reject":
		status = db.TicketRejected
	case "close":
		status = db.TicketClosed
	default:
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	row, err := h.deps.DB.AdminTicket(ctx, siteID(ctx), body.ID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-page-not-found")))
		return
	}
	if !h.ticketsAllowed(w, r, loc, row.Kind) {
		return
	}
	mine := auth.FromContext(ctx)
	if mine == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	// Role grants only make sense when approving a membership application,
	// and only admins with ManagePermissions may pick one. Otherwise the
	// stored grant (if any) is preserved, like the old admin screen.
	role := row.GrantedID
	if body.Action == "approve" && row.Kind == db.MembershipApplyKind && body.GrantedRole != nil && *body.GrantedRole > 0 {
		subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(mine, time.Now())
		if err != nil {
			h.deps.log().Error("resolve permissions", "user", mine.ID, "err", err)
			writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
			return
		}
		if perms.Resolve(subject, nil).Has(perms.ManagePermissions) {
			role = body.GrantedRole
		}
	}
	note := strings.TrimSpace(body.Note)
	if len(note) > 2000 {
		note = note[:2000]
	}
	// The reply endpoint owns the reply text; handling preserves it.
	if err := h.deps.DB.ReviewTicket(ctx, siteID(ctx), row.ID, status, note, row.Reply, mine.ID, role, time.Now()); err != nil {
		h.deps.log().Error("handle ticket", "id", row.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	h.tellSubmitter(ctx, row, mine.ID, status, row.Reply)
	writeOK(w, loc)
}
