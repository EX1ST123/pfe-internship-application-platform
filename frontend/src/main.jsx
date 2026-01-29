import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import ApplyPage from "./App";
import HrBackoffice from "./HRBackoffice";
import { AuthProvider } from "./AuthContext";
import "./styles.css";

ReactDOM.createRoot(document.getElementById("root")).render(
  <AuthProvider>
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<ApplyPage />} />
        <Route path="/backoffice" element={<HrBackoffice />} />
      </Routes>
    </BrowserRouter>
  </AuthProvider>
);
