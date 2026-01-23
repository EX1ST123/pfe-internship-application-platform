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
import { Link } from "react-router-dom";

export default function HrBackoffice() {
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
    application_type: ""
  });

  useEffect(() => {
    // Fetch all applications
    fetch("http://localhost:8080/applications")
      .then(res => res.json())
      .then(setApplications)
      .catch(err => console.error("Failed to fetch applications:", err));
    
    // Fetch weekly applications count
    fetch("http://localhost:8080/weekly-applications")
      .then(res => res.json())
      .then(data => setWeeklyCount(data.count))
      .catch(err => console.error("Failed to fetch weekly count:", err));
  }, []);

  /* ================= DERIVED DATA ================= */

  const filtered = useMemo(() => {
    return applications.filter(a => {
      const searchMatch =
        `${a.full_name} ${a.email}`.toLowerCase().includes(search.toLowerCase());
      const degreeMatch = filters.degree_level
        ? a.degree_level === filters.degree_level
        : true;
      const typeMatch = filters.application_type
        ? a.application_type === filters.application_type
        : true;

      return searchMatch && degreeMatch && typeMatch;
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

  /* ================= RENDER ================= */

  return (
    <main>
      <header className="header">
        <div className="logo">
          <BsMortarboardFill /> PFE Portal
        </div>
        <Link to="/" className="btn-primary">
          Apply now
        </Link>
      </header>

      {/* TITLE */}
      <div className="hr-title">
        <h1>HR Backoffice</h1>
        <p>Manage and review all internship applications</p>
      </div>

      {/* STATS */}
      <div className="stats">
        <Stat icon={<BsPeopleFill />} label="Total Applications" value={applications.length} />
        <Stat
          icon={<BsMortarboardFill />}
          label="Engineering Degree"
          value={applications.filter(a => a.degree_level === "Engineering").length}
        />

        <Stat icon={<BsClockFill />} label="This Week" value={weeklyCount} />
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
          <button
            type="button"
            className="filter-btn"
            onClick={() => setFiltersVisible(!filtersVisible)}
          >
            <BsFilter /> Filters
          </button>
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
      </div>

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
                  <span className={`pill ${a.application_type.toLowerCase()}`}>
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
                        className="action-link"
                      >
                        View
                      </a>
                      <a
                        href={`http://localhost:8080/${a.cv_file_path}`}
                        download
                        className="action-link"
                      >
                        Download
                      </a>
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

function Stat({ icon, label, value }) {
  return (
    <div className="stat">
      <div className="stat-icon">{icon}</div>
      <div>
        <strong>{value}</strong>
        <span>{label}</span>
      </div>
    </div>
  );
}