export type ContainerSummary = {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  labels: Record<string, string>;
  // Docker API returns Names as /name, so we handle that in UI or here
  names?: string[]; 
};

// Alias for compatibility
export type Container = ContainerSummary;

export type RestartPolicyName = "no" | "always" | "on-failure" | "unless-stopped";

export type RestartPolicy = {
  name: RestartPolicyName;
  maximumRetryCount: number;
};

export type RemediationAction = "none" | "start" | "restart";

export type MonitorType = "http" | "container";

export type MonitorConfig = {
  id: string;
  name: string;
  type: MonitorType;
  intervalSeconds: number;
  timeoutSeconds: number;
  notifyWebhookIds: string[];
  createdAt: string;
  updatedAt: string;
  http?: { url: string };
  container?: {
    containerId: string;
    restartPolicy?: RestartPolicy;
    remediation: { action: RemediationAction; maxAttempts: number; cooldownSeconds: number };
  };
  logs: { include: boolean; tail: number };
};

// Enhanced monitor with status for UI
export type Monitor = MonitorConfig & {
  status: string; // up, down, pending
  lastCheck?: string;
  url?: string; // Helper for display
  containerName?: string; // Helper for display
};

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(path, init);
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(text || `HTTP ${resp.status}`);
  }
  const ct = resp.headers.get("content-type") ?? "";
  if (ct.includes("application/json")) {
    return (await resp.json()) as T;
  }
  return (await resp.text()) as T;
}

export const listContainers = () => req<ContainerSummary[]>("/api/containers");
export const getContainers = listContainers;

export const startContainer = (id: string) => req<{ ok: boolean }>(`/api/containers/${id}/start`, { method: "POST" });

export const stopContainer = (id: string, timeoutSeconds = 10) =>
  req<{ ok: boolean }>(`/api/containers/${id}/stop`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ timeoutSeconds })
  });

export const restartContainer = (id: string, timeoutSeconds = 10) =>
  req<{ ok: boolean }>(`/api/containers/${id}/restart`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ timeoutSeconds })
  });

export const setRestartPolicy = (id: string, policy: string) =>
  req<{ ok: boolean }>(`/api/containers/${id}/restart-policy`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ Name: policy, MaximumRetryCount: 0 })
  });

export const getContainerLogs = async (id: string, tail = 200) => {
  const resp = await fetch(`/api/containers/${id}/logs?tail=${encodeURIComponent(String(tail))}`);
  if (!resp.ok) {
    throw new Error(await resp.text());
  }
  return await resp.text();
};

export const listMonitors = () => req<MonitorConfig[]>("/api/monitors");
export const getMonitors = listMonitors;

export type NotificationWebhook = {
  id?: string;
  name: string;
  url: string;
  type: string;
  editable?: boolean;
};

export const getNotifications = () => req<NotificationWebhook[]>("/api/notifications");

export const createNotification = (n: NotificationWebhook) =>
  req<NotificationWebhook>("/api/notifications", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(n) });

export const deleteNotification = (id: string) => req<{ ok: boolean }>(`/api/notifications/${id}`, { method: "DELETE" });

export const createMonitor = (m: Partial<MonitorConfig> & Pick<MonitorConfig, "name" | "type">) =>
  req<MonitorConfig>("/api/monitors", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(m) });

export const updateMonitor = (id: string, m: Partial<MonitorConfig> & Pick<MonitorConfig, "name" | "type">) =>
  req<MonitorConfig>(`/api/monitors/${id}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(m) });

export const deleteMonitor = (id: string) => req<{ ok: boolean }>(`/api/monitors/${id}`, { method: "DELETE" });

export type MonitorStatusInfo = {
  status: string;
  lastCheck: string;
};

export const getStatus = () => req<{ status: Record<string, MonitorStatusInfo> }>("/api/status");

// Helper to merge data
export async function getMonitorsWithStatus(): Promise<Monitor[]> {
  const [monitors, statusData] = await Promise.all([listMonitors(), getStatus()]);
  return monitors.map(m => {
    const info = statusData.status[m.id];
    const status = info ? info.status : 'pending';
    const lastCheck = info ? info.lastCheck : undefined;
    
    let url = '';
    let containerName = '';
    
    if (m.type === 'http' && m.http) url = m.http.url;
    if (m.type === 'container' && m.container) containerName = m.container.containerId;

    return {
      ...m,
      status,
      url,
      containerName,
      lastCheck
    };
  });
}
