import { Navigate, Route, Routes, useLocation } from "react-router-dom";

import Navbar from "./components/navbar";
import Footer from "./components/footer";
import HomePage from "./pages/HomePage";
import GalleryPage from "./pages/GalleryPage";
import AboutPage from "./pages/AboutPage";
import ShareViewPage from "./pages/ShareViewPage";
import AdminLayout from "./pages/admin/AdminLayout";
import AdminDashboardPage from "./pages/admin/AdminDashboardPage";
import AdminConfigPage from "./pages/admin/AdminConfigPage";
import AdminPasswordPage from "./pages/admin/AdminPasswordPage";

function App() {
  const location = useLocation();
  const isAdmin = location.pathname.startsWith("/admin");
  const className = isAdmin ? "min-h-screen" : 
    "container mx-auto box-border max-w-7xl px-3 md:px-6";
  const style = isAdmin ? { minHeight: "var(--app-height)" } : 
    { minHeight: "var(--main-height)", paddingTop: "36px" };

  return (<>
    {!isAdmin && <Navbar />}
    <main 
      className={className}
      style={style}
    >
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/gallery" element={<GalleryPage />} />
        <Route path="/about" element={<AboutPage />} />
        <Route path="/s/:code" element={<ShareViewPage />} />
        <Route element={<AdminLayout />} path="/admin">
          <Route index element={<Navigate replace to="/admin/dashboard" />} />
          <Route element={<AdminPasswordPage />} path="password" />
          <Route element={<AdminDashboardPage />} path="dashboard" />
          <Route element={<AdminConfigPage />} path="config" />
        </Route>
        <Route path="*" element={<HomePage />} />
      </Routes>
    </main>
    {!isAdmin && <Footer />}
  </>);
}

export default App;
