import React, { useEffect, useMemo, useState } from "react";
import {
  BsPeopleFill,
  BsMortarboardFill,
  BsClockFill,
  BsBarChartFill,
  BsSearch,
  BsArrowDownUp,
  BsChevronLeft,
  BsChevronRight,
  BsFilter
} from "react-icons/bs";
import "./hr-backoffice.css";
import { Link, useNavigate } from "react-router-dom";

export default function HrBackoffice() {
  const navigate = useNavigate();
  const [applications, setApplications] = useState([]);
  const [weeklyCount, setWeeklyCount] = useState(0); 
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState("created_at");
  const [sortDir, setSortDir] = useState("desc");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(5);
  const [filtersVisible, setFiltersVisible] = useState(false);
  const [filters, setFilters] = useState({
    degree_level: "",
    application_type: "",
    this_week: false,
  });
  const [showModal, setShowModal] = useState(false);
  const [newSubject, setNewSubject] = useState("");
  const [successMsg, setSuccessMsg] = useState("");
  const [subjects, setSubjects] = useState([]);
  const [deleteModal, setDeleteModal] = useState(false);
  const [selectedSubjects, setSelectedSubjects] = useState([]);
  // EDIT SUBJECT
  const [editModal, setEditModal] = useState(false);
  const [editSubjectId, setEditSubjectId] = useState(null);
  const [editSubjectName, setEditSubjectName] = useState("");
  // USER STATE
  const [user, setUser] = useState(null);

  useEffect(() => {
    // Fetch user info
    fetch("http://localhost:8080/me", {
      credentials: "include"
    })
      .then(res => res.json())
      .then(data => {
        if (data.loggedIn) {
          setUser(data);
        }
      })
      .catch(() => {});

    // Fetch all applications
    fetch("http://localhost:8080/applications", {
      credentials: "include"
    })
      .then(res => res.json())
      .then(data => {
        if (Array.isArray(data)) {
          setApplications(data);
        } else {
          setApplications([]);
        }
      })
      .catch(() => setApplications([]));

    
    // Fetch weekly applications count
    fetch("http://localhost:8080/weekly-applications", {
      credentials: "include"
    })
      .then(res => res.json())
      .then(data => setWeeklyCount(data.count))
      .catch(err => console.error("Failed to fetch weekly count:", err));
  }, []);
  
useEffect(() => {
  fetchSubjects();
}, []);


  const handleLogout = async () => {
    await fetch("http://localhost:8080/logout", {
      method: "POST",
      credentials: "include",
    });
    setUser(null);
    alert("Logged out successfully");
    navigate("/");
  };
  
  const getStartOfWeek = () => {
      const now = new Date();
      const day = now.getDay(); // 0 = Sunday
      const diff = now.getDate() - day + (day === 0 ? -6 : 1);
      return new Date(now.setDate(diff));
    };

  /* ================= DERIVED DATA ================= */

const filtered = useMemo(() => {
  const startOfWeek = getStartOfWeek();

  return applications.filter(a => {
    const searchMatch =
      `${a.full_name} ${a.email}`.toLowerCase().includes(search.toLowerCase());

    const degreeMatch = filters.degree_level
      ? a.degree_level === filters.degree_level
      : true;

    const typeMatch = filters.application_type
      ? a.application_type === filters.application_type
      : true;

    const weekMatch = filters.this_week
      ? new Date(a.created_at) >= startOfWeek
      : true;

    return searchMatch && degreeMatch && typeMatch && weekMatch;
  });
}, [applications, search, filters]);

  const sorted = useMemo(() => {
    return [...filtered].sort((a, b) => {
      const v1 = a[sortKey];
      const v2 = b[sortKey];
      if (v1 < v2) return sortDir === "asc" ? -1 : 1;
      if (v1 > v2) return sortDir === "asc" ? 1 : -1;
      return 0;
    });
  }, [filtered, sortKey, sortDir]);

  const totalPages = Math.ceil(sorted.length / pageSize);
  const paginated = sorted.slice(
    (page - 1) * pageSize,
    page * pageSize
  );

  /* ================= HANDLERS ================= */

  const toggleSort = (key) => {
    if (sortKey === key) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const handleFilterChange = (key, value) => {
    setPage(1);
    setFilters({ ...filters, [key]: value });
  };

  const fetchSubjects = async () => {
  try {
    const res = await fetch("http://localhost:8080/subjects", {
      credentials: "include",
    });
    const data = await res.json();
    setSubjects(Array.isArray(data) ? data : []);
  } catch {
    setSubjects([]);
  }
};

  const handleDownloadCV = async (cvPath, applicantName) => {
    try {
      const response = await fetch(`http://localhost:8080/${cvPath}`, {
        credentials: "include"
      });
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `CV_${applicantName.replace(/\s+/g, '_')}.pdf`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Download failed:', error);
      alert('Failed to download CV');
    }
  };

  /* ================= RENDER ================= */

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
          <Link to="/" className="btn-primary">
            Application form
          </Link>
        </div>
      </header>

      {/* TITLE */}
      <div className="hr-title">
        <h1>HR Backoffice</h1>
        <p>Manage and review all internship applications</p>
      </div>

      {/* STATS */}
      <div className="stats">
        <Stat
          icon={<BsPeopleFill />}
          label="Total Applications"
          value={applications.length}
          variant="primary"
        />

        <Stat
          icon={<BsMortarboardFill />}
          label="Engineering Degree"
          value={applications.filter(a => a.degree_level === "Engineering").length}
          variant="success"
        />

        <Stat
          icon={<BsClockFill />}
          label="This Week"
          value={weeklyCount}
          variant="info"
        />

        <Stat
          icon={<BsBarChartFill />}
          label="Solo Applications"
          value={
            applications.length
              ? Math.round(
                  (applications.filter(a => a.application_type === "Solo").length /
                    applications.length) * 100
                ) + "%"
              : "0%"
          }
          variant="warning"
        />
      </div>


      {/* SEARCH + FILTER */}
      <div className="toolbar">
        <div className="search-group">
        
          <input
            placeholder="🔍Search by name or email..."
            value={search}
            onChange={(e) => {
              setPage(1);
              setSearch(e.target.value);
            }}
          />
         
        </div>

        <div className="page-size">
          <label>
            Show{" "}
            <select
              value={pageSize}
              onChange={(e) => {
                setPage(1);
                setPageSize(Number(e.target.value));
              }}
            >
              {[5, 10, 20, 50].map(size => (
                <option key={size} value={size}>{size}</option>
              ))}
            </select>
          </label>
        </div>
        <button
          className="btn-primary"
          onClick={() => setShowModal(true)}
        >
          Add Subject
        </button>
      <button
        className="btn-edit"
        onClick={async () => {
          await fetchSubjects();
          setEditModal(true);
        }}
      >
        Edit Subject
      </button>

       <button
          className="btn-delete"
          onClick={async () => {
            await fetchSubjects();
            setDeleteModal(true);
          }}
        >
          Delete Subject
        </button>
               <button
            type="button"
            className="filter-btn"
            onClick={() => setFiltersVisible(!filtersVisible)}
          >
            <BsFilter /> Filters
          </button>
      </div>
      {editModal && (
          <div className="modal-overlay">
            <div className="modal">
              <h3>Edit Subject</h3>

              <div className="subjects">
                {subjects.map(s => (
                  <label key={s.id} className="subject-pill">
                    <input
                      type="radio"
                      checked={editSubjectId === s.id}
                      onChange={() => {
                        setEditSubjectId(s.id);
                        setEditSubjectName(s.name);
                      }}
                    />
                    {s.name}
                  </label>
                ))}
              </div>

              <input
                placeholder="New subject name"
                value={editSubjectName}
                onChange={(e) => setEditSubjectName(e.target.value)}
                disabled={!editSubjectId}
              />

              <div className="modal-actions">
                <button
                  className="btn-secondary"
                  onClick={() => {
                    setEditModal(false);
                    setEditSubjectId(null);
                    setEditSubjectName("");
                  }}
                >
                  Cancel
                </button>

                <button
                  className="btn-primary"
                  onClick={async () => {
                    if (!editSubjectId || !editSubjectName.trim()) return;

                    const res = await fetch("http://localhost:8080/subjects", {
                      method: "PUT",
                      headers: { "Content-Type": "application/json" },
                      body: JSON.stringify({
                        id: editSubjectId,
                        name: editSubjectName
                      }),
                    });

                    if (res.ok) {
                      setSubjects(prev =>
                        prev.map(s =>
                          s.id === editSubjectId
                            ? { ...s, name: editSubjectName }
                            : s
                        )
                      );
                      alert("Subject updated successfully");
                      setEditModal(false);
                    } else {
                      alert("Subject name already exists");
                    }
                  }}
                >
                  Save
                </button>
              </div>
            </div>
          </div>
        )}

      {deleteModal && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Delete Subjects</h3>

            <div className="subjects">
              {subjects.map(s => (
                <label key={s.id} className="subject-pill">
                  <input
                    type="checkbox"
                    checked={selectedSubjects.includes(s.id)}
                    onChange={(e) => {
                      setSelectedSubjects(prev =>
                        e.target.checked
                          ? [...prev, s.id]
                          : prev.filter(id => id !== s.id)
                      );
                    }}
                  />
                  {s.name}
                </label>
              ))}
            </div>

            <div className="modal-actions">
              <button
                className="btn-secondary"
                onClick={() => {
                  setDeleteModal(false);
                  setSelectedSubjects([]);
                }}
              >
                Cancel
              </button>

              <button
                className="btn-primary"
                onClick={async () => {
                  if (selectedSubjects.length === 0) return;

                  if (!window.confirm(
                    "Are you sure you want to delete the selected subjects?"
                  )) return;

                  const res = await fetch("http://localhost:8080/subjects/delete", {
                    method: "DELETE",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ ids: selectedSubjects }),
                  });

                  if (res.ok) {
                    setSubjects(prev =>
                      prev.filter(s => !selectedSubjects.includes(s.id))
                    );
                    setSelectedSubjects([]);
                    setDeleteModal(false);
                    alert("Subjects deleted");
                  } else {
                    alert("Some subjects are in use and cannot be deleted");
                  }
                }}
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {showModal && (
  <div className="modal-overlay">
    <div className="modal">
      <h3>Add New Subject</h3>

      {successMsg && (
        <div className="success-box">{successMsg}</div>
      )}

      <input
        placeholder="Subject name"
        value={newSubject}
        onChange={(e) => setNewSubject(e.target.value)}
      />

      <div className="modal-actions">
        <button
          className="btn-secondary"
          onClick={() => {
            setShowModal(false);
            setNewSubject("");
            setSuccessMsg("");
          }}
        >
          Cancel
        </button>

        <button
            className="btn-primary"
            onClick={async () => {
              if (!newSubject.trim()) return;

              const res = await fetch("http://localhost:8080/subjects", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name: newSubject }),
              });

              if (res.ok) {
                setSuccessMsg("Subject added successfully");
                setNewSubject("");
                await fetchSubjects();

              } else {
                alert("Subject already exists");
              }
            }}
          >
            Submit
          </button>
        </div>
      </div>
    </div>
  )}

      {/* FILTER DROPDOWN */}
      {filtersVisible && (
        <div className="filters-dropdown">
          <label>
            Degree:
            <select
              value={filters.degree_level}
              onChange={(e) =>
                handleFilterChange("degree_level", e.target.value)
              }
            >
              <option value="">All</option>
              <option value="Engineering">Engineering</option>
              <option value="Bachelor">Bachelor</option>
              <option value="Master">Master</option>
            </select>
          </label>

          <label>
            Type:
            <select
              value={filters.application_type}
              onChange={(e) =>
                handleFilterChange("application_type", e.target.value)
              }
            >
              <option value="">All</option>
              <option value="Solo">Solo</option>
              <option value="Pair">Pair</option>
            </select>
          </label>
          <label className="checkbox-filter">
            <input
              type="checkbox"
              checked={filters.this_week}
              onChange={(e) => {
                setPage(1);
                setFilters({ ...filters, this_week: e.target.checked });
              }}
            />
            This week only
          </label>

        </div>
      )}


      <div className="table-card">
        <table>
          <thead>
            <tr>
              <th onClick={() => toggleSort("created_at")}>
                Date <BsArrowDownUp />
              </th>
              <th onClick={() => toggleSort("full_name")}>
                Name <BsArrowDownUp />
              </th>
              <th>Email</th>
              <th>Degree</th>
              <th>Type</th>
              <th>Start Date</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {paginated.map(a => (
              <tr key={a.id}>
                <td>{a.created_at}</td>
                <td>{a.full_name}</td>
                <td className="muted">{a.email}</td>
                <td><span className="badge">{a.degree_level}</span></td>
                <td>
                  <span className={`pill ${(a.application_type || "").toLowerCase()}`}>

                    {a.application_type}
                  </span>
                </td>
                <td>{a.start_date || "N/A"}</td>
                <td className="actions">
                  {a.cv_file_path ? (
                    <>
                      <a
                        href={`http://localhost:8080/${a.cv_file_path}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="btn-primary"
                      >
                        View
                      </a>
                      <button
                        onClick={() => handleDownloadCV(a.cv_file_path, a.full_name)}
                        className="btn-edit"
                        
                      >
                        Download
                      </button>
                    </>
                  ) : (
                    <span className="muted">No CV</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {/* PAGINATION */}
        <div className="pagination">
          <button disabled={page === 1} onClick={() => setPage(page - 1)}>
            <BsChevronLeft />
          </button>
          <span>Page {page} of {totalPages}</span>
          <button disabled={page === totalPages} onClick={() => setPage(page + 1)}>
            <BsChevronRight />
          </button>
        </div>
      </div>
    </main>
  );
}

function Stat({ icon, label, value, variant = "primary" }) {
  return (
    <div className="stat">
      <div className={`stat-group stat-${variant}`}>
        <div className="stat-icon">{icon}</div>
        <strong className="stat-value">{value}</strong>
      </div>
      <span>{label}</span>
    </div>
  );
}
