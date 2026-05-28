import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { clearApiBaseUrl, getApiBaseUrl, setApiBaseUrl } from "@/lib/config";
import { api } from "@/lib/api";
import { useLogLevel, useLogLevelMutation } from "@/hooks/useRouterSync";
import { useEffect, useState } from "react";
import { ExternalLink } from "lucide-react";

export function SettingsPage() {
  const [url, setUrl] = useState(() => getApiBaseUrl());
  const [testResult, setTestResult] = useState<string | null>(null);
  const [testing, setTesting] = useState(false);
  const logLevel = useLogLevel();
  const setLogLevel = useLogLevelMutation();
  const [level, setLevel] = useState("warning");

  useEffect(() => {
    if (logLevel.data?.level) setLevel(logLevel.data.level);
  }, [logLevel.data?.level]);

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

  const applyLogLevel = () => {
    setLogLevel.mutate(level, {
      onSuccess: (data) => {
        setTestResult(`Log level set to ${data.level}`);
      },
      onError: (e) => {
        setTestResult(`Log level failed — ${(e as Error).message}`);
      },
    });
  };

  const swaggerUrl = url ? `${url.replace(/\/$/, "")}/swagger/index.html` : "/swagger/index.html";
  const levels = logLevel.data?.levels ?? [
    "trace",
    "debug",
    "info",
    "warning",
    "error",
    "fatal",
    "panic",
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Settings</h1>
        <p className="text-sm text-muted-foreground">
          API connection and runtime service options.
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
                  <code className="rounded bg-muted px-1">.env.development.local</code>). Active:{" "}
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
                  setTestResult("Cleared saved URL — using Vite proxy. Reload the page.");
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
          <CardTitle>Logging verbosity</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 max-w-xl">
          <p className="text-xs text-muted-foreground">
            Changes apply immediately on the router-sync process (no restart). Reverts to config
            file level on restart.
          </p>
          <div className="flex flex-wrap items-end gap-3">
            <div className="min-w-[160px] flex-1">
              <Label>Log level</Label>
              <Select
                value={level}
                onChange={(e) => setLevel(e.target.value)}
                disabled={logLevel.isLoading || setLogLevel.isPending}
              >
                {levels.map((l) => (
                  <option key={l} value={l}>
                    {l}
                  </option>
                ))}
              </Select>
            </div>
            <Button onClick={applyLogLevel} disabled={setLogLevel.isPending || logLevel.isLoading}>
              Apply
            </Button>
          </div>
          {logLevel.data && (
            <p className="text-xs text-muted-foreground">
              Current on server: <strong>{logLevel.data.level}</strong>
            </p>
          )}
          {logLevel.isError && (
            <p className="text-sm text-destructive">
              Could not load log level — redeploy router-sync if this endpoint is missing.
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
