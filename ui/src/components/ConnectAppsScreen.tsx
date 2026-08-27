import { useEffect, useState } from "react";
import { getIntegrationStatuses, type IntegrationStatus } from "@/api";
import { Switch } from "@/components/ui/switch";
import { ClaudeAppSettings } from "@/components/ClaudeAppSettings";

const ICONS: Record<string, string> = {
  "claude-desktop": "/launch-icons/claude.svg",
  "claude": "/launch-icons/claude-code.svg",
  "codex": "/launch-icons/codex.svg",
  "opencode": "/launch-icons/opencode.svg",
  "copilot": "/launch-icons/copilot.svg",
  "chatgpt": "/launch-icons/codex-app.png",
};

function AppIcon({ id }: { id: string }) {
  const src = ICONS[id];
  if (src) {
    return <img src={src} alt="" className="h-7 w-7 object-contain" />;
  }
  return (
    <span className="flex h-7 w-7 items-center justify-center rounded bg-neutral-100 text-[11px] font-semibold text-neutral-600">
      {id.slice(0, 2).toUpperCase()}
    </span>
  );
}

function CommandCopyControl({
  command,
  name,
  isCopied,
  onCopy,
}: {
  command: string;
  name: string;
  isCopied: boolean;
  onCopy: () => void;
}) {
  return (
    <div className="flex shrink-0 items-center gap-2">
      <code className="rounded-lg bg-neutral-100 px-3 py-1.5 font-mono text-[13px] leading-none text-neutral-700">{command}</code>
      <button
        type="button"
        onClick={onCopy}
        aria-label={isCopied ? "Copied" : `Copy ${name} command`}
        title={isCopied ? "Copied" : "Copy"}
        className="inline-flex h-7 w-7 items-center justify-center rounded-md text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700"
      >
        {isCopied ? (
          <svg width="14" height="14" viewBox="0 0 20 20" fill="none"><path d="M6 10l2 2 5-5" stroke="#16a34a" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round"/><rect x="3.5" y="3.5" width="13" height="13" rx="2" stroke="#16a34a" strokeWidth="1.2"/></svg>
        ) : (
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none"><rect x="3" y="3" width="9" height="9" rx="1.2" stroke="currentColor" strokeWidth="1.1"/><path d="M6 3.5V2.8A1.2 1.2 0 0 1 7.2 1.6H12.4A1.2 1.2 0 0 1 13.6 2.8V8a1.2 1.2 0 0 1-1.2 1.2H11" stroke="currentColor" strokeWidth="1.1"/></svg>
        )}
      </button>
    </div>
  );
}

export function ConnectAppsScreen() {
  const [statuses, setStatuses] = useState<IntegrationStatus[] | null>(null);
  const [error, setError] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const [claudeEnabled, setClaudeEnabled] = useState(true);

  useEffect(() => {
    getIntegrationStatuses().then(setStatuses).catch(() => setError(true));
  }, []);
  useEffect(() => {
    if (!copied) return;
    const t = setTimeout(() => setCopied(null), 2000);
    return () => clearTimeout(t);
  }, [copied]);
  const copy = (command: string) => {
    void navigator.clipboard.writeText(command).then(() => setCopied(command));
  };

  if (error) return <p className="p-8 text-sm text-red-600">Couldn&apos;t load integrations.</p>;
  if (!statuses) return <p className="p-8 text-sm text-neutral-400">Checking integrations…</p>;

  const claudeDesktop = statuses.find(s => s.id === "claude-desktop");
  const chatGPTDesktop = statuses.find(s => s.id === "chatgpt");
  const chatGPTCommand = chatGPTDesktop?.command;
  const terminalOrder = ["claude", "codex", "opencode", "copilot"];
  const byId = new Map(statuses.map(s => [s.id, s] as const));
  const terminalOrdered: Array<IntegrationStatus & { command: string }> = [];
  for (const id of terminalOrder) {
    const it = byId.get(id);
    if (it?.command) terminalOrdered.push({ ...it, command: it.command });
  }

  return (
    <main className="flex min-h-0 w-full flex-1 flex-col overflow-y-auto bg-white text-neutral-950">
      <div className="mx-auto w-full max-w-[880px] px-6 pb-8 pt-6">
        <h1 className="text-[11px] font-semibold uppercase tracking-widest text-neutral-500">Apps</h1>

        <section className="mt-6">
          <h2 className="text-[11px] font-semibold uppercase tracking-widest text-neutral-400">Desktop</h2>
          {claudeDesktop && (
            <>
              <div className="mt-3 flex items-center justify-between gap-4 py-3">
                <div className="flex min-w-0 items-center gap-3">
                  <img src={ICONS["claude-desktop"]} alt="" className="h-8 w-8 object-contain" />
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-neutral-900">Claude</p>
                    <p className="text-sm leading-5 text-neutral-500">{claudeDesktop.description}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span className="hidden sm:inline text-sm text-neutral-600">{claudeEnabled ? "Hide" : ""}</span>
                  <Switch label="Show Claude Desktop settings" checked={claudeEnabled} onCheckedChange={setClaudeEnabled} />
                </div>
              </div>
              {claudeEnabled && (
                <div className="rounded-xl border border-neutral-200 bg-white overflow-hidden">
                  <ClaudeAppSettings />
                </div>
              )}
            </>
          )}
          {chatGPTDesktop && (
            <div className="mt-3 flex items-center justify-between gap-4 py-3">
              <div className="flex min-w-0 items-center gap-3">
                <AppIcon id="chatgpt" />
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-neutral-900">{chatGPTDesktop.name}</p>
                  <p className="text-sm leading-5 text-neutral-500">{chatGPTDesktop.description}</p>
                </div>
              </div>
              {chatGPTCommand && (
                <CommandCopyControl
                  command={chatGPTCommand}
                  name={chatGPTDesktop.name}
                  isCopied={copied === chatGPTCommand}
                  onCopy={() => copy(chatGPTCommand)}
                />
              )}
            </div>
          )}
        </section>

        <section className="mt-8">
          <h2 className="text-[11px] font-semibold uppercase tracking-widest text-neutral-400">Terminal</h2>
          <div className="mt-3 flex flex-col">
            {terminalOrdered.map(item => {
              const isCopied = copied === item.command;
              return (
                <div key={item.id} className="flex flex-col gap-1 py-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
                  <div className="flex min-w-0 items-center gap-3">
                    <AppIcon id={item.id} />
                    <div className="min-w-0">
                      <p className="text-sm font-semibold text-neutral-900">{item.name}</p>
                      <p className="truncate text-sm leading-5 text-neutral-500">{item.description}</p>
                    </div>
                  </div>
                  <CommandCopyControl
                    command={item.command}
                    name={item.name}
                    isCopied={isCopied}
                    onCopy={() => item.command && copy(item.command)}
                  />
                </div>
              );
            })}
          </div>
        </section>
      </div>
    </main>
  );
}
