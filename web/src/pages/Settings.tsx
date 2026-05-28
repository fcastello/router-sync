import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { clearApiBaseUrl, getApiBaseUrl, setApiBaseUrl } from "@/lib/config";
import { api } from "@/lib/api";
import {
  useLogLevels,
  useServiceLogLevelMutation,
} from "@/hooks/useRouterSync";
import { useEffect, useMemo, useState } from "react";
import { ExternalLink } from "lucide-react";

export function SettingsPage() {
  const [url, setUrl] = useState(() => getApiBaseUrl());
  const [testResult, setTestResult] = useState<string | null>(null);
  const [testing, setTesting] = useState(false);

  const logLevels = useLogLevels(10000);
  const setLogLevel = useServiceLogLevelMutation();

  const [pending, setPending] = useState<Record<string, string>>({});

  const services = useMemo(() => {
    const all = logLevels.data?.services ?? {};
    const keys = Object.keys(all).sort((a, b) => {
      if (a === "api") return -1;
      if (b === "api") return 1;
      return a.localeCompare(b);
    });
    return keys.map((id) => ({ id, ...all[id] }));
  }, [logLevels.data]);

  useEffect(() => {
    const next: Record<string, string> = {};
    services.forEach((s) => {
      if (!pending[s.id]) next[s.id] = s.level;
    });
    if (Object.keys(next).length > 0) {
      setPending((prev) => ({ ...next, ...prev }));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [logLevels.data]);

  const availableLevels = logLevels.data?.levels ?? [
    "trace",
    "debug",
    "info",
    "warning",
    "error",
    "fatal",
    "panic",
  ];

  const save = () => {
    setApiBaseUrl(url);
    setTestResult("Saved. Reload the page if requests still use the old URL.");
  };

  const test = async () => {
    setTesting(true);
    setTestResult(null);
    const prev = getApiBaseUrl();
    setApiBaseUrl(url);
    try {
      const h = await api.health();
      setTestResult(`OK — ${h.service} (${h.status})`);
    } catch (e) {
      setTestResult(`Failed — ${(e as Error).message}`);
      setApiBaseUrl(prev);
    } finally {
      setTesting(false);
    }
  };

  const apply = (serviceId: string) => {
    const level = pending[serviceId];
    if (!level) return;
    setLogLevel.mutate(
      { serviceId, level },
      {
        onSuccess: () => {
          setTestResult(`Set ${serviceId} → ${level}`);
        },
        onError: (e) => {
          setTestResult(
            `Failed to set ${serviceId} — ${(e as Error).message}`,
          );
        },
      },
    );
  };

  const swaggerUrl = url
    ? `${url.replace(/\/$/, "")}/swagger/index.html`
    : "/swagger/index.html";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Settings</h1>
        <p className="text-sm text-muted-foreground">
          API connection and per-service runtime options.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>API base URL</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 max-w-xl">
          <div>
            <Label>URL (no trailing slash)</Label>
            <Input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="http://192.168.2.252:18080"
            />
            <p className="mt-1 text-xs text-muted-foreground">
              {import.meta.env.DEV ? (
                <>
                  In dev, leave empty and use the Vite proxy to R2 (see{" "}
                  <code className="rounded bg-muted px-1">
                    .env.development.local
                  </code>
                  ). Active:{" "}
                  <code className="rounded bg-muted px-1">
                    {getApiBaseUrl() || "proxy → R2"}
                  </code>
                </>
              ) : (
                <>
                  Current:{" "}
                  <code className="rounded bg-muted px-1">
                    {getApiBaseUrl() || "(not set)"}
                  </code>
                </>
              )}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button onClick={save}>Save</Button>
            <Button variant="outline" onClick={test} disabled={testing}>
              Test connection
            </Button>
            {import.meta.env.DEV && (
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  clearApiBaseUrl();
                  setUrl("");
                  setTestResult(
                    "Cleared saved URL — using Vite proxy. Reload the page.",
                  );
                }}
              >
                Use dev proxy
              </Button>
            )}
          </div>
          {testResult && <p className="text-sm">{testResult}</p>}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Logging verbosity per service</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-xs text-muted-foreground">
            Changes apply at runtime (no restart). The API watches{" "}
            <code className="rounded bg-muted px-1">level.api</code> and each
            agent watches{" "}
            <code className="rounded bg-muted px-1">
              level.agent.&lt;hostname&gt;
            </code>
            .
          </p>

          {services.length === 0 && (
            <p className="text-sm text-muted-foreground">
              {logLevels.isLoading
                ? "Loading services…"
                : "No services reporting yet. Make sure the API and at least one agent are running."}
            </p>
          )}

          <div className="space-y-2">
            {services.map((svc) => (
              <div
                key={svc.id}
                className="flex flex-wrap items-center gap-3 rounded-md border border-border px-3 py-2"
              >
                <div className="min-w-[150px]">
                  <div className="font-mono text-sm">{svc.id}</div>
                  <div className="text-[11px] text-muted-foreground">
                    current: {svc.level || "?"}
                  </div>
                </div>
                <Badge variant={svc.online ? "success" : "warn"}>
                  {svc.online ? "online" : svc.source || "stale"}
                </Badge>
                <div className="ml-auto flex items-end gap-2">
                  <Select
                    value={pending[svc.id] ?? svc.level ?? "warning"}
                    onChange={(e) =>
                      setPending((p) => ({ ...p, [svc.id]: e.target.value }))
                    }
                    disabled={setLogLevel.isPending}
                  >
                    {availableLevels.map((l) => (
                      <option key={l} value={l}>
                        {l}
                      </option>
                    ))}
                  </Select>
                  <Button
                    onClick={() => apply(svc.id)}
                    disabled={
                      setLogLevel.isPending ||
                      (pending[svc.id] ?? svc.level) === svc.level
                    }
                  >
                    Apply
                  </Button>
                </div>
              </div>
            ))}
          </div>

          {logLevels.isError && (
            <p className="text-sm text-destructive">
              Could not load log levels — redeploy router-sync if the endpoint
              is missing.
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>API docs</CardTitle>
        </CardHeader>
        <CardContent>
          <a
            href={swaggerUrl}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
          >
            Open Swagger UI
            <ExternalLink className="h-3 w-3" />
          </a>
        </CardContent>
      </Card>
    </div>
  );
}
