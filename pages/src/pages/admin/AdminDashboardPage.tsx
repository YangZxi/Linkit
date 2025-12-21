import { useCallback, useEffect, useState } from "react";
import { Button, Card, CardBody, CardHeader, Spinner, addToast } from "@heroui/react";

import api from "@/lib/api";

type AdminDashboardStats = {
  totalFiles: number;
  totalFileSize: number;
  totalShareViews: number;
};

const emptyStats: AdminDashboardStats = {
  totalFiles: 0,
  totalFileSize: 0,
  totalShareViews: 0,
};

function formatBytes(size: number) {
  if (!Number.isFinite(size) || size <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let idx = 0;
  let value = size;
  while (value >= 1024 && idx < units.length - 1) {
    value /= 1024;
    idx += 1;
  }
  const digits = idx === 0 || value >= 10 ? 0 : 1;
  return `${value.toFixed(digits)} ${units[idx]}`;
}

export default function AdminDashboardPage() {
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<AdminDashboardStats>(emptyStats);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get<AdminDashboardStats>("/admin/stats");
      setStats(res ?? emptyStats);
    } catch (err) {
      setStats(emptyStats);
      addToast({
        title: "加载失败",
        description: (err as Error).message,
        color: "danger",
        variant: "flat",
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-default-900">面板</h1>
          <p className="mt-1 text-sm text-default-500">查看整体数据统计。</p>
        </div>
        <Button color="primary" isLoading={loading} onPress={load}>
          刷新
        </Button>
      </div>
      <Card className="border border-default-200/70 bg-background/70 shadow-sm">
        <CardHeader className="flex items-center justify-between">
          <div>
            <p className="text-base font-semibold">数据统计</p>
          </div>
        </CardHeader>
        <CardBody className="space-y-4">
          {loading ? (
            <div className="flex items-center justify-center py-10">
              <Spinner label="加载中..." />
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-3">
              <div className="rounded-lg border border-default-200/60 bg-background/60 p-4">
                <p className="text-xs text-default-500">文件总数</p>
                <p className="mt-2 text-2xl font-semibold text-default-900">
                  {stats.totalFiles.toLocaleString()}
                </p>
              </div>
              <div className="rounded-lg border border-default-200/60 bg-background/60 p-4">
                <p className="text-xs text-default-500">文件总大小</p>
                <p className="mt-2 text-2xl font-semibold text-default-900">
                  {formatBytes(stats.totalFileSize)}
                </p>
                <p className="mt-1 text-xs text-default-400">
                  {stats.totalFileSize.toLocaleString()} B
                </p>
              </div>
              <div className="rounded-lg border border-default-200/60 bg-background/60 p-4">
                <p className="text-xs text-default-500">分享访问次数</p>
                <p className="mt-2 text-2xl font-semibold text-default-900">
                  {stats.totalShareViews.toLocaleString()}
                </p>
              </div>
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
