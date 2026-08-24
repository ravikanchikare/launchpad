package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type AccountSummary struct {
	TeamID        string  `json:"teamId"`
	TeamAlias     string  `json:"teamAlias,omitempty"`
	Role          string  `json:"role,omitempty"`
	UserEmail     string  `json:"userEmail,omitempty"`
	Spend         float64 `json:"spend"`
	MaxBudget     float64 `json:"maxBudget,omitempty"`
	BudgetResetAt string  `json:"budgetResetAt,omitempty"`
	RPMLimit      *int    `json:"rpmLimit,omitempty"`
}

type userInfoResponse struct {
	UserID   string `json:"user_id"`
	UserInfo struct {
		Teams     []string `json:"teams"`
		UserEmail string   `json:"user_email"`
	} `json:"user_info"`
}

type memberMeResponse struct {
	UserID             string   `json:"user_id"`
	TeamID             string   `json:"team_id"`
	TeamAlias          string   `json:"team_alias"`
	Role               string   `json:"role"`
	UserEmail          string   `json:"user_email"`
	Spend              *float64 `json:"spend"`
	LitellmBudgetTable *struct {
		MaxBudget     *float64 `json:"max_budget"`
		RPMLimit      *int     `json:"rpm_limit"`
		BudgetResetAt *string  `json:"budget_reset_at"`
	} `json:"litellm_budget_table"`
}

func (c *Client) UserTeams(ctx context.Context) ([]string, error) {
	var out userInfoResponse
	if err := c.doJSON(ctx, http.MethodGet, "/user/info", nil, &out); err != nil {
		return nil, err
	}
	return out.UserInfo.Teams, nil
}

func (c *Client) MemberMe(ctx context.Context, teamID string) (AccountSummary, error) {
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return AccountSummary{}, fmt.Errorf("team id is required")
	}
	var out memberMeResponse
	path := "/team/" + teamID + "/members/me"
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return AccountSummary{}, err
	}
	summary := AccountSummary{
		TeamID:    out.TeamID,
		TeamAlias: out.TeamAlias,
		Role:      out.Role,
		UserEmail: out.UserEmail,
	}
	if out.Spend != nil {
		summary.Spend = *out.Spend
	}
	if out.LitellmBudgetTable != nil {
		if out.LitellmBudgetTable.MaxBudget != nil {
			summary.MaxBudget = *out.LitellmBudgetTable.MaxBudget
		}
		if out.LitellmBudgetTable.RPMLimit != nil {
			summary.RPMLimit = out.LitellmBudgetTable.RPMLimit
		}
		if out.LitellmBudgetTable.BudgetResetAt != nil {
			summary.BudgetResetAt = *out.LitellmBudgetTable.BudgetResetAt
		}
	}
	return summary, nil
}

func (c *Client) AccountSummary(ctx context.Context) (AccountSummary, error) {
	teams, err := c.UserTeams(ctx)
	if err != nil {
		return AccountSummary{}, err
	}
	if len(teams) == 0 {
		return AccountSummary{}, fmt.Errorf("no team membership found for this key")
	}
	return c.MemberMe(ctx, teams[0])
}
