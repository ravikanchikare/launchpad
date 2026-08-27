import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { getSettings, updateSettings, getLauncherConfig, updateLauncherConfig } from "@/api";
import { Switch } from "@/components/ui/switch";

export function SettingsScreen() {
  const qc = useQueryClient();
  const { data, isLoading, error } = useQuery({ queryKey: ["settings"], queryFn: getSettings });
  const { data: launcherConfig } = useQuery({ queryKey: ["launcher-config"], queryFn: getLauncherConfig });
  const [providerUrl, setProviderUrl] = useState("");

  useEffect(() => {
    if (!launcherConfig) return;
    setProviderUrl(launcherConfig.providerUrl);
  }, [launcherConfig]);

  const mut = useMutation({
    mutationFn: updateSettings,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["settings"] }),
  });
  const launcherMut = useMutation({
    mutationFn: updateLauncherConfig,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["launcher-config"] }),
  });

  if (isLoading) return <div className="p-8 text-sm text-neutral-400">Loading settings…</div>;
  if (error || !data) return <div className="p-8 text-sm text-red-600">Couldn&apos;t load settings.</div>;

  const s = data.settings;

  return (
    <main className="flex min-h-0 w-full flex-1 flex-col overflow-y-auto bg-white p-6">
      <div className="mx-auto w-full max-w-2xl space-y-6">
        <div>
          <h1 className="px-4 text-lg font-medium text-neutral-950">Settings</h1>
          <p className="mt-1 px-4 text-xs leading-5 text-neutral-500">
            Configure the LiteLLM provider and application behavior.
          </p>
        </div>

        <section className="rounded-xl border border-neutral-200 bg-white overflow-hidden">
          <div className="border-b border-neutral-200 px-4 py-3.5">
            <h2 className="text-sm font-medium text-neutral-950">LiteLLM provider</h2>
            <p className="text-xs leading-5 text-neutral-500">
              The CLI automatically uses <code>LITELLM_API_KEY</code> when it is available.
            </p>
          </div>
          <form
            className="space-y-4 px-4 py-4"
            onSubmit={(event) => {
              event.preventDefault();
              launcherMut.mutate({ providerUrl });
            }}
          >
            <label className="block">
              <span className="text-xs font-medium text-neutral-700">Provider URL</span>
              <input
                value={providerUrl}
                onChange={(event) => setProviderUrl(event.target.value)}
                className="mt-1 w-full rounded-lg border border-neutral-200 px-3 py-2 text-sm outline-none focus:border-neutral-400"
                placeholder="http://localhost:4000"
                type="url"
                required
              />
            </label>
            <div className="flex items-center justify-between gap-4">
              <p className="text-xs text-neutral-500">
                API key: {launcherConfig?.apiKeyConfigured ? "detected" : "not detected"}
              </p>
              <button
                type="submit"
                disabled={launcherMut.isPending}
                className="rounded-lg bg-neutral-900 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
              >
                {launcherMut.isPending ? "Saving…" : "Save provider"}
              </button>
            </div>
            {launcherMut.error && <p className="text-xs text-red-600">{launcherMut.error.message}</p>}
          </form>
        </section>

        <div className="divide-y divide-neutral-200 rounded-xl border border-neutral-200 overflow-hidden bg-white">
          <div className="flex items-center justify-between gap-4 px-4 py-4">
            <div className="min-w-0">
              <p className="text-sm font-medium text-neutral-950">Show apps in menu</p>
              <p className="text-xs leading-5 text-neutral-500">Show Launchpad in the system menu bar / tray</p>
            </div>
            <Switch
              label="Show apps in menu"
              checked={!!s.showAppsInMenu}
              onCheckedChange={(v) => mut.mutate({ showAppsInMenu: v })}
            />
          </div>
          <div className="flex items-center justify-between gap-4 px-4 py-4">
            <div className="min-w-0">
              <p className="text-sm font-medium text-neutral-950">Auto-download updates</p>
              <p className="text-xs leading-5 text-neutral-500">Download and stage updates in the background</p>
            </div>
            <Switch
              label="Auto-download updates"
              checked={!!s.autoUpdateEnabled}
              onCheckedChange={(v) => mut.mutate({ autoUpdateEnabled: v })}
            />
          </div>
        </div>
        <p className="px-4 text-[11px] leading-4 text-neutral-400">
          Provider keys are read from <code>LITELLM_API_KEY</code> or the macOS Keychain and are never stored in this settings file.
        </p>
      </div>
    </main>
  );
}
