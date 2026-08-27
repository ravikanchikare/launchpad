const API_BASE = "";

export interface Settings {
  showAppsInMenu: boolean;
  autoUpdateEnabled: boolean;
}

export async function getSettings(): Promise<{ settings: Settings }> {
  const r = await fetch(`${API_BASE}/api/v1/settings`);
  if (!r.ok) throw new Error("settings fetch failed");
  return r.json();
}

export async function updateSettings(s: Partial<Settings>): Promise<{ settings: Settings }> {
  const r = await fetch(`${API_BASE}/api/v1/settings`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(s),
  });
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

export interface LauncherConfig {
  providerUrl: string;
  cliName: string;
  apiKeyConfigured: boolean;
}

export async function getLauncherConfig(): Promise<LauncherConfig> {
  const r = await fetch(`${API_BASE}/api/v1/launcher/config`);
  if (!r.ok) throw new Error("launcher config fetch failed");
  return r.json();
}

export async function updateLauncherConfig(s: Pick<LauncherConfig, "providerUrl">): Promise<LauncherConfig> {
  const r = await fetch(`${API_BASE}/api/v1/launcher/config`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(s),
  });
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

export interface IntegrationStatus {
  id: string;
  name: string;
  description: string;
  installed?: boolean;
  command?: string;
}

export async function getIntegrationStatuses(): Promise<IntegrationStatus[]> {
  const r = await fetch(`${API_BASE}/api/v1/integrations`);
  if (!r.ok) throw new Error("integrations fetch failed");
  return r.json();
}

export async function launchHarness(id: string) {
  const r = await fetch(`${API_BASE}/api/v1/harnesses/${id}/launch`, { method: "POST" });
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}
