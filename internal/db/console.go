package db

import (
	"context"
	"fmt"
)

// ConsoleOverview is the dashboard card set for the /-/console admin shell.
type ConsoleOverview struct {
	Users          int
	Articles       int
	PendingReports int
	PendingTickets int
}

var qConsoleUserCount = register("ConsoleUserCount", `SELECT count(*) FROM web_user`)

var qConsoleArticleCount = register("ConsoleArticleCount", `SELECT count(*) FROM web_article WHERE site_id = $1`)

var qConsolePendingReports = register("ConsolePendingReports",
	`SELECT count(*) FROM web_userreport WHERE status = $1 AND site_id = $2`)

var qConsolePendingTickets = register("ConsolePendingTickets",
	`SELECT count(*) FROM web_userticket WHERE status = $1 AND site_id = $2`)

// ConsoleOverview reads the headline numbers for the console dashboard.
func (d *DB) ConsoleOverview(ctx context.Context, siteID int64) (ConsoleOverview, error) {
	var out ConsoleOverview
	if err := d.pool.QueryRow(ctx, qConsoleUserCount).Scan(&out.Users); err != nil {
		return out, fmt.Errorf("count console users: %w", err)
	}
	if err := d.pool.QueryRow(ctx, qConsoleArticleCount, siteID).Scan(&out.Articles); err != nil {
		return out, fmt.Errorf("count console articles: %w", err)
	}
	if err := d.pool.QueryRow(ctx, qConsolePendingReports, ReportPending, siteID).Scan(&out.PendingReports); err != nil {
		return out, fmt.Errorf("count console reports: %w", err)
	}
	if err := d.pool.QueryRow(ctx, qConsolePendingTickets, TicketPending, siteID).Scan(&out.PendingTickets); err != nil {
		return out, fmt.Errorf("count console tickets: %w", err)
	}
	return out, nil
}
