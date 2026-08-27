import { useEffect, useState } from "react";
import { Switch } from "@/components/ui/switch";

type ClaudeConfig = {
  fable_5: string;
  opus_5: string;
  sonnet_5: string;
  haiku_4_5: string;
  sonnet_4_6: string;
  autoMode: boolean;
  running: boolean;
};

type ClaudeModelKey = "fable_5" | "opus_5" | "sonnet_5" | "haiku_4_5" | "sonnet_4_6";

const ROWS: { key: ClaudeModelKey; label: string }[] = [
  { key: "fable_5", label: "Fable 5" },
  { key: "opus_5", label: "Opus 5" },
  { key: "sonnet_5", label: "Sonnet 5" },
  { key: "haiku_4_5", label: "Haiku 4.5" },
  { key: "sonnet_4_6", label: "Sonnet 4.6" },
];

export function ClaudeAppSettings() {
  const [cfg, setCfg] = useState<ClaudeConfig | null>(null);
  const [models, setModels] = useState<string[]>([]);
  const [modelsError, setModelsError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [restarting, setRestarting] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/v1/apps/claude").then(r => r.json()).then(setCfg).catch(() => setCfg({ fable_5: "", opus_5: "", sonnet_5: "", haiku_4_5: "", sonnet_4_6: "", autoMode: false, running: false }));
    fetch("/api/v1/apps/claude/models")
      .then(async response => {
        if (!response.ok) throw new Error((await response.text()).trim() || "Could not load gateway models");
        return response.json();
      })
      .then(setModels)
      .catch(error => setModelsError(error instanceof Error ? error.message : "Could not load gateway models"));
  }, []);

  const update = async (patch: Partial<ClaudeConfig>) => {
    if (!cfg) return;
    const next = { ...cfg, ...patch };
    setCfg(next);
    setSaving(true);
    try {
      const r = await fetch("/api/v1/apps/claude", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) });
      if (!r.ok) throw new Error();
      setMsg(null);
    } catch {
      setMsg("Failed to save");
    } finally {
      setSaving(false);
    }
  };

  const updateModel = (key: ClaudeModelKey, model: string) => {
    void update({ [key]: model });
  };

  const restart = async () => {
    if (cfg?.running && !window.confirm("Restart Claude Desktop? Any running task will stop.")) return;
    setRestarting(true);
    try {
      const response = await fetch("/api/v1/apps/claude/restart", { method: "POST" });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Failed to start Claude Desktop");
      setCfg(current => current ? { ...current, running: true } : current);
      setMsg("Claude Desktop is running");
      setTimeout(() => setMsg(null), 2500);
    } catch (error) {
      setMsg(error instanceof Error ? error.message : "Failed to start Claude Desktop");
    } finally {
      setRestarting(false);
    }
  };

  const reset = async () => {
    await fetch("/api/v1/apps/claude/reset", { method: "POST" });
    setCfg(current => ({ fable_5: "", opus_5: "", sonnet_5: "", haiku_4_5: "", sonnet_4_6: "", autoMode: false, running: current?.running ?? false }));
    setMsg("Reset to defaults");
    setTimeout(()=>setMsg(null), 2000);
  };

  if (!cfg) return <div className="px-6 py-6 text-sm text-neutral-400">Loading Claude settings…</div>;

  return (
    <div className="px-6 py-6">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-[15px] font-semibold text-neutral-950">Claude</h3>
            <p className="mt-1 max-w-[520px] text-sm leading-5 text-neutral-500">Choose which Launchpad model Claude uses for each model option.</p>
          </div>
          <button
            type="button"
            onClick={restart}
            disabled={restarting}
            className="shrink-0 rounded-lg border border-neutral-200 bg-white px-3.5 py-1.5 text-sm font-medium text-neutral-800 hover:bg-neutral-50 disabled:opacity-50"
          >
            {restarting ? (cfg.running ? "Restarting…" : "Starting…") : (cfg.running ? "Restart Claude" : "Start Claude")}
          </button>
        </div>

        <div className="mt-6 flex flex-col gap-3">
          {modelsError && <p className="text-xs text-red-600">{modelsError}</p>}
          {ROWS.map(row => (
            <div key={row.key} className="grid grid-cols-[140px_16px_1fr] items-center gap-3">
              <span className="text-sm font-medium text-neutral-900">{row.label}</span>
              <span className="text-center text-neutral-400">→</span>
              <div className="relative">
                <select
                  value={cfg[row.key]}
                  onChange={event => updateModel(row.key, event.target.value)}
                  className="w-full appearance-none rounded-lg border border-neutral-200 bg-neutral-50 px-3 py-2 pr-8 text-sm text-neutral-800 placeholder:text-neutral-400 focus:border-neutral-300 focus:outline-none focus:ring-0"
                >
                  <option value="">Select a model</option>
                  {models.map(m => (
                    <option key={m} value={m}>{m}</option>
                  ))}
                  {cfg[row.key] && !models.includes(cfg[row.key]) && (
                    <option value={cfg[row.key]}>{cfg[row.key]}</option>
                  )}
                </select>
                <span className="pointer-events-none absolute inset-y-0 right-2 flex items-center text-neutral-400">
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M4 6l4 4 4-4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round"/></svg>
                </span>
              </div>
            </div>
          ))}
        </div>

        <div className="mt-6 flex items-center justify-between gap-4 border-t border-neutral-200 pt-5">
          <div>
            <p className="text-sm font-medium text-neutral-900">Enable auto mode</p>
            <p className="text-sm leading-5 text-neutral-500">Let Claude decide when to ask before making changes.</p>
          </div>
          <Switch label="Enable Claude auto mode" checked={!!cfg.autoMode} onCheckedChange={v => update({ autoMode: v })} />
        </div>

        {msg && <p className="mt-3 text-xs text-green-600">{msg}</p>}
        {saving && <p className="mt-1 text-xs text-neutral-400">Saving…</p>}

        <div className="mt-6 flex justify-end">
          <button
            type="button"
            onClick={reset}
            className="rounded-lg border border-neutral-200 bg-white px-3.5 py-1.5 text-sm font-medium text-neutral-800 hover:bg-neutral-50"
          >
            Reset to defaults
          </button>
        </div>
      </div>
  );
}
