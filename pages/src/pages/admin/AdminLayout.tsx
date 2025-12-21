import { Outlet, Link as RouterLink, useLocation } from "react-router-dom";
import { Button } from "@heroui/react";
import clsx from "clsx";
import api from "@/lib/api";
import { UserProfile } from "@/types/user";
import { useEffect, useState } from "react";

const navItems = [
  { label: "面板", href: "/admin/dashboard" },
  { label: "配置管理", href: "/admin/config" },
  { label: "修改密码", href: "/admin/password" },
  { label: "主页", href: "/" },
] as const;

export default function AdminLayout() {
  const location = useLocation();
  const [user, setUser] = useState<UserProfile | null>(null);

  useEffect(() => {
    const check = async () => {
      const u = await api.get<UserProfile>("/me").then((data) => data).catch(() => null);
      if (!u) {
        window.location.href = "/";
        return null;
      }
      setUser(u);
    }
    check()
  }, []);

  if (!user) {
    return null;
  }

  return (
    <div className="flex min-h-screen w-full bg-background">
      <aside className="w-64 border-r border-default-200/60 bg-background/80 px-4 py-6 backdrop-blur">
        <div className="px-2 pb-4">
          <p className="text-sm font-semibold text-default-900">后台管理</p>
        </div>
        <nav className="flex flex-col gap-2">
          {navItems.map((item) => {
            const active = location.pathname === item.href;
            return (
              <Button
                key={item.href}
                as={RouterLink}
                className={clsx("justify-start", active && "bg-default-100")}
                color={active ? "primary" : "default"}
                to={item.href}
                variant={active ? "flat" : "light"}
              >
                {item.label}
              </Button>
            );
          })}
        </nav>
      </aside>

      <section className="flex min-w-0 flex-1 flex-col">
        <div className="mx-auto w-full max-w-5xl px-6 py-8">
          <Outlet />
        </div>
      </section>
    </div>
  );
}

