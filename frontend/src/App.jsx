import React, { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  BsMortarboardFill,
  BsPersonFill,
  BsPeopleFill,
  BsMortarboard,
  BsBriefcaseFill,
  BsBookFill,
  BsFileEarmarkTextFill,
  BsCheckCircleFill,
  BsUpload,
} from "react-icons/bs";

export default function ApplyPage() {
  const navigate = useNavigate();
  const [applicationType, setApplicationType] = useState("Solo");
  const [cvName, setCvName] = useState("");
  const [motivationName, setMotivationName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [subjects, setSubjects] = useState([]);
  /* ===== AUTH STATE ===== */
  const [user, setUser] = useState(null);
  const [showAuth, setShowAuth] = useState(false);
  const [authMode, setAuthMode] = useState("login");
  const [authForm, setAuthForm] = useState({ role: "user" });

  useEffect(() => {
    fetch("http://localhost:8080/subjects")
      .then(res => res.json())
      .then(setSubjects)
      .catch(() => setSubjects([]));

    // Check if user is logged in
    fetch("http://localhost:8080/me", {
      credentials: "include",
    })
      .then(res => res.json())
      .then(data => {
        if (data.loggedIn) {
          setUser(data);
        }
      })
      .catch(() => {});
  }, []);

  const validateForm = async (formData) => {
    // Validate phone number (8 digits)
    const phone = formData.get("phone") || "";
    const phoneDigits = phone.replace(/\D/g, '');
    if (phoneDigits.length !== 8) {
      alert("❌ Phone number must be exactly 8 digits");
      return false;
    }

    // Validate email format
    const email = formData.get("email") || "";
    if (!email.includes("@") || !email.includes(".") || email.indexOf("@") > email.lastIndexOf(".")) {
      alert("❌ Please enter a valid email address (example@domain.com)");
      return false;
    }
    
    const res = await fetch(
      `http://localhost:8080/email-exists?email=${encodeURIComponent(email)}`
    );
    const data = await res.json();

    if (data.exists) {
      alert("❌ Email already used");
      return false;
    }

    // Validate at least one subject selected
    const subjects = formData.getAll("subjects");
    if (subjects.length === 0) {
      alert("❌ Please select at least one subject");
      return false;
    }

    // Validate PDF files
    const cvFile = document.querySelector('input[name="cv"]').files[0];
    if (cvFile && cvFile.type !== "application/pdf") {
      alert("❌ CV must be a PDF file");
      return false;
    }

    const motivationFile = document.querySelector('input[name="motivation"]').files[0];
    if (motivationFile && motivationFile.type !== "application/pdf") {
      alert("❌ Motivation letter must be a PDF file");
      return false;
    }
    
    return true;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitted(false);

    const form = e.target;
    const formData = new FormData(form);

    // Validate form before submission
    if (!(await validateForm(formData))) {
      return;
    }

    try {
      setSubmitting(true);

      const res = await fetch("http://localhost:8080/apply", {
        method: "POST",
        body: formData,
      });

      const responseText = await res.text();

      if (!res.ok) {
        alert("❌ " + (responseText || "Submission failed"));
        return;
      }

      alert("✅ Application submitted successfully!");
      setSubmitted(true);
      form.reset();
      setCvName("");
      setMotivationName("");
      setApplicationType("Solo");
      
    } catch (err) {
      alert("❌ " + (err.message || "Submission failed"));
    } finally {
      setSubmitting(false);
    }
  };

  const submitAuth = async () => {
    const res = await fetch(
      `http://localhost:8080/${authMode}`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(authForm),
      }
    );

    if (!res.ok) {
      alert("❌ Authentication failed");
      return;
    }

    const me = await fetch("http://localhost:8080/me", {
      credentials: "include",
    });
    const data = await me.json();
    setUser(data);
    setShowAuth(false);
    alert("✅ Success");
    
    // If user is admin, redirect to backoffice
    if (data.role === "admin") {
      navigate("/backoffice");
    }
  };

  const handleBackofficeClick = () => {
    if (user && user.role === "admin") {
      navigate("/backoffice");
    } else {
      setAuthMode("login");
      setShowAuth(true);
    }
  };

  const handleLogout = async () => {
    await fetch("http://localhost:8080/logout", {
      method: "POST",
      credentials: "include",
    });
    setUser(null);
    alert("✅ Logged out successfully");
  };

  return (
    <main>
      <header className="header">
        <div className="logo">
          <BsMortarboardFill /> PFE Portal
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
          {user && user.loggedIn && (
            <>
              <span style={{ marginRight: '0.5rem' }}>
                <strong>{user.username || 'User'}</strong>
              </span>
              <button onClick={handleLogout} className="btn-secondary">
                Logout
              </button>
            </>
          )}
          {!user || !user.loggedIn ? (
            <button onClick={handleBackofficeClick} className="btn-primary">
              Log in
            </button>
          ) : user.role === "admin" ? (
            <button onClick={handleBackofficeClick} className="btn-primary">
              HR Backoffice
            </button>
          ) : null}
        </div>
      </header>

      <section className="hero">
        <span className="badge">Applications Open for 2026</span>
        <h1>
          Start Your Career Journey <br />
          with a <span>PFE Internship</span>
        </h1>
      </section>

      <form className="container" onSubmit={handleSubmit}>
        <h2>Submit Your Application</h2>

        <div className="card">
          <h3><BsPersonFill /> Personal Information</h3>
          <div className="grid-2">
            <input name="full_name" placeholder="Full Name *" required />
            <div className="radio-group">
              <label><input type="radio" name="gender" value="Male" required /> Male</label>
              <label><input type="radio" name="gender" value="Female" /> Female</label>
            </div>
            <input name="email" placeholder="Email *" required />
            <input name="phone" placeholder="Phone *" required />
          </div>
        </div>

        <div className="card">
          <h3><BsPeopleFill /> Application Type</h3>
          <div className="grid-2">
            {["Solo", "Pair"].map((type) => (
              <label key={type} className={`select-card ${applicationType === type ? "active" : ""}`}>
                <input
                  type="radio"
                  name="application_type"
                  value={type}
                  checked={applicationType === type}
                  onChange={() => setApplicationType(type)}
                />
                <strong>{type}</strong>
              </label>
            ))}
          </div>
        </div>

        <div className="card">
          <h3><BsMortarboard /> Academic Information</h3>
          <div className="grid-2">
            <input name="university" placeholder="University *" required />
            <input name="field_of_study" placeholder="Field of Study *" required />
            <select name="degree_level" required>
              <option value="">Degree Level *</option>
              <option>Bachelor</option>
              <option>Master</option>
              <option>Engineering</option>
            </select>
          </div>
        </div>

        <div className="card">
          <h3><BsBriefcaseFill /> Internship Preferences</h3>
          <div className="grid-2">
            <select name="internship_duration" required>
              <option value="">Internship Duration *</option>
              <option>4 months</option>
              <option>6 months</option>
            </select>
            <select name="preferred_working_method" required>
              <option value="">Preferred Working Method *</option>
              <option>Onsite</option>
              <option>Remote</option>
              <option>Hybrid</option>
            </select>
            <input 
              type="date" 
              name="early_start_date" 
              required 
            />
          </div>
        </div>

        <div className="card">
          <h3><BsBookFill /> Application Subjects *</h3>
          <div className="subjects">
            {subjects.map(s => (
              <label key={s.id} className="subject-pill">
                <input type="checkbox" name="subjects" value={s.name} />
                {s.name}
              </label>
            ))}
          </div>
        </div>

        <div className="card">
          <h3><BsFileEarmarkTextFill /> Documents</h3>
          <div className="grid-2">
            <label className="upload-box">
              <input type="file" name="cv" accept="application/pdf" required
                onChange={(e) => setCvName(e.target.files[0]?.name || "")} />
              {cvName ? <><BsCheckCircleFill /> {cvName}</> : <><BsUpload /> Upload CV</>}
            </label>

            <label className="upload-box">
              <input type="file" name="motivation" accept="application/pdf"
                onChange={(e) => setMotivationName(e.target.files[0]?.name || "")} />
              {motivationName ? <><BsCheckCircleFill /> {motivationName}</> : <><BsUpload /> Motivation Letter</>}
            </label>
          </div>
        </div>

        <button className="btn-submit" disabled={submitting}>
          {submitting ? "Submitting..." : "Submit Application"}
        </button>
      </form>

      {showAuth && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>{authMode === "signup" ? "Sign Up" : "Login"}</h3>

            <input
              placeholder="Username"
              onChange={e => setAuthForm({ ...authForm, username: e.target.value })}
            />

            {authMode === "signup" && (
              <>
                <input
                  placeholder="Email"
                  onChange={e => setAuthForm({ ...authForm, email: e.target.value })}
                />
                <select
                  onChange={e => setAuthForm({ ...authForm, role: e.target.value })}
                >
                  <option value="user">Student</option>
                  <option value="admin">Admin</option>
                </select>
              </>
            )}

            <input
              type="password"
              placeholder="Password"
              onChange={e => setAuthForm({ ...authForm, password: e.target.value })}
            />

            <button className="btn-primary" onClick={submitAuth}>
              Submit
            </button>

            <p
              style={{ cursor: "pointer", marginTop: 10 }}
              onClick={() =>
                setAuthMode(authMode === "signup" ? "login" : "signup")
              }
            >
              {authMode === "signup"
                ? "Already have an account?"
                : "Don't have an account?"}
            </p>

            <button
              style={{ marginTop: 10 }}
              className="btn-secondary"
              onClick={() => setShowAuth(false)}
            >
              Close
            </button>
          </div>
        </div>
      )}
    </main>
  );
}