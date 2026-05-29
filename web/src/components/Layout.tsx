import { cn } from "@/lib/utils";
import { Activity, Cpu, LayoutDashboard, Network, Route, Server, Settings } from "lucide-react";
import { NavLink, Outlet } from "react-router-dom";
import { useHealth } from "@/hooks/useRouterSync";
import { Badge } from "@/components/ui/badge";

const nav = [
  { to: "/", label: "Policies", icon: Route },
  { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { to: "/routers", label: "Routers", icon: Cpu },
  { to: "/devices", label: "Devices", icon: Network },
  { to: "/providers", label: "Providers", icon: Server },
  { to: "/settings", label: "Settings", icon: Settings },
];

export function Layout() {
  const health = useHealth();

  const online = health.data?.status === "ok" || health.data?.status === "healthy";

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-56 flex-col border-r border-border bg-card">
        <div className="border-b border-border px-4 py-4">
          <div className="flex items-center gap-2 font-semibold">
            <Activity className="h-5 w-5 text-primary" />
            Router Sync
          </div>
          <div className="mt-2 flex items-center gap-2">
            <span
              className={cn(
                "h-2 w-2 rounded-full",
                online ? "bg-green-500" : health.isError ? "bg-red-500" : "bg-amber-400",
              )}
            />
            <span className="text-xs text-muted-foreground">
              API {online ? "online" : health.isLoading ? "checking…" : "offline"}
            </span>
          </div>
        </div>
        <nav className="flex flex-1 flex-col gap-1 p-2">
          {nav.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              end={to === "/"}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2 rounded-md px-3 py-2 text-sm transition",
                  isActive
                    ? "bg-primary/10 font-medium text-primary"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground",
                )
              }
            >
              <Icon className="h-4 w-4" />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="border-t border-border p-3">
          <Badge variant={online ? "success" : "warn"} className="w-full justify-center">
            {health.data?.service ?? "router-sync"}
          </Badge>
        </div>
      </aside>
      <main className="flex-1 overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  );
}
