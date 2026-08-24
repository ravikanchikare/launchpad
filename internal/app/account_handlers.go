package app

import (
	"context"

	"harnezpad/internal/gateway"
)

func (m *Manager) AccountSummary(ctx context.Context) (gateway.AccountSummary, error) {
	if err := m.RequireGatewayToken(); err != nil {
		return gateway.AccountSummary{}, err
	}
	return m.gatewayClient().AccountSummary(ctx)
}

func (m *Manager) ListModelCatalog(ctx context.Context) ([]gateway.ModelCatalogEntry, error) {
	if err := m.RequireGatewayToken(); err != nil {
		return nil, err
	}
	return m.gatewayClient().ListModelGroups(ctx)
}
