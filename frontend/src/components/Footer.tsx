"use client";

import { useEffect, useState } from "react";

// Injected at build time from the local .env (never committed). Images built
// without it (e.g. in CI) simply don't show the password.
const GRAFANA_PASSWORD = process.env.NEXT_PUBLIC_GRAFANA_PASSWORD ?? "";
const GRAFANA_PORT = process.env.NEXT_PUBLIC_GRAFANA_PORT ?? "8081";

export function Footer() {
  const [grafanaUrl, setGrafanaUrl] = useState<string | null>(null);

  useEffect(() => {
    const { protocol, hostname } = window.location;
    setGrafanaUrl(`${protocol}//${hostname}:${GRAFANA_PORT}`);
  }, []);

  return (
    <footer className="border-t border-slate-200 bg-white">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center gap-x-4 gap-y-1 px-4 py-4 text-sm text-slate-600">
        {grafanaUrl && (
          <a href={grafanaUrl} target="_blank" rel="noopener noreferrer" className="text-indigo-600 hover:underline">
            Grafana
          </a>
        )}
        {GRAFANA_PASSWORD && (
          <span>
            Contraseña: <code className="rounded bg-slate-100 px-1.5 py-0.5 font-mono">{GRAFANA_PASSWORD}</code>
          </span>
        )}
      </div>
    </footer>
  );
}
