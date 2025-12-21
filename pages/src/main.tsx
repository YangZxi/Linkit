import React, { useEffect } from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, useNavigate } from "react-router-dom";
import { HeroUIProvider, ToastProvider } from "@heroui/react";

import App from "./App.tsx";
import "@/styles/globals.css";


function Provider({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate();

  useEffect(() => {
    const updateAppHeight = () => {
      // 使用真实可视高度，避免移动端地址栏导致 100vh 误差
      document.documentElement.style.setProperty(
        "--app-height",
        `${window.innerHeight}px`,
      );
    };

    updateAppHeight();
    window.addEventListener("resize", updateAppHeight);
    window.addEventListener("orientationchange", updateAppHeight);

    return () => {
      window.removeEventListener("resize", updateAppHeight);
      window.removeEventListener("orientationchange", updateAppHeight);
    };
  }, []);

  return <HeroUIProvider navigate={navigate}
    className="w-full min-h-screen theme-wrapper"
    style={{
      backgroundImage: `url()`,
      backgroundSize: "cover",
      backgroundRepeat: "no-repeat",
      backgroundPosition: "center",
      backgroundAttachment: "fixed",
      minHeight: "var(--app-height)",
    }}
  >
    <ToastProvider placement="top-right" />
    {children}
  </HeroUIProvider>;
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <BrowserRouter>
      <Provider>
        <App />
      </Provider>
    </BrowserRouter>
  </React.StrictMode>,
);
