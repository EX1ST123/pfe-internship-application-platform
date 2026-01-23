import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import ApplyPage from "./App";
import HrBackoffice from "./HrBackoffice";
import "./styles.css";

ReactDOM.createRoot(document.getElementById("root")).render(
  <BrowserRouter>
    <Routes>
      <Route path="/" element={<ApplyPage />} />
      <Route path="/backoffice" element={<HrBackoffice />} />
    </Routes>
  </BrowserRouter>
);
