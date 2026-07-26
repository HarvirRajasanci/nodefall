import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AuthProvider } from "./AuthContext";
import LoginPage from "./pages/LoginPage";
import RegisterPage from "./pages/RegisterPage";
import HomePage from "./pages/HomePage";
import QueuePage from "./pages/QueuePage";
import PlayPage from "./pages/PlayPage";
import FriendsPage from "./pages/FriendsPage";

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/queue" element={<QueuePage />} />
          <Route path="/play" element={<PlayPage />} />
          <Route path="/friends" element={<FriendsPage />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
