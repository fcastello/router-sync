import { Layout } from "@/components/Layout";
import { Route, Routes } from "react-router-dom";
import { DashboardPage } from "@/pages/Dashboard";
import { DevicesPage } from "@/pages/Devices";
import { PoliciesPage } from "@/pages/Policies";
import { ProvidersPage } from "@/pages/Providers";
import { RoutersPage } from "@/pages/Routers";
import { SettingsPage } from "@/pages/Settings";

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<PoliciesPage />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="routers" element={<RoutersPage />} />
        <Route path="devices" element={<DevicesPage />} />
        <Route path="policies" element={<PoliciesPage />} />
        <Route path="providers" element={<ProvidersPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  );
}
