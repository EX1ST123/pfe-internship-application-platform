package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var store = sessions.NewCookieStore([]byte("super-secret-key"))

type ApplicationResponse struct {
	ID                     int      `json:"id"`
	FullName               string   `json:"full_name"`
	Email                  string   `json:"email"`
	Gender                 string   `json:"gender"`
	Phone                  string   `json:"phone"`
	University             string   `json:"university"`
	FieldOfStudy           string   `json:"field_of_study"`
	DegreeLevel            string   `json:"degree_level"`
	ApplicationType        string   `json:"application_type"`
	InternshipDuration     string   `json:"internship_duration"`
	PreferredWorkingMethod string   `json:"preferred_working_method"`
	StartDate              *string  `json:"start_date,omitempty"`
	CreatedAt              string   `json:"created_at"`
	CVFilePath             string   `json:"cv_file_path"`
	MotivationFilePath     *string  `json:"motivation_file_path,omitempty"`
	Subjects               []string `json:"subjects"`
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, GET, POST, DELETE, OPTIONS")
}

func deleteSubjects(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var payload struct {
		IDs []int `json:"ids"`
	}
	json.NewDecoder(r.Body).Decode(&payload)

	if len(payload.IDs) == 0 {
		http.Error(w, "No subjects selected", 400)
		return
	}

	db.Exec(`DELETE FROM application_subjects WHERE subject_id = ANY($1)`, pq.Array(payload.IDs))
	db.Exec(`DELETE FROM subjects WHERE id = ANY($1)`, pq.Array(payload.IDs))

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func authRequired(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		session, _ := store.Get(r, "auth")

		userRole, ok := session.Values["role"].(string)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if role != "" && userRole != role {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func main() {
	var err error
	db, err = sql.Open("postgres",
		"host=localhost port=5432 user=postgres password=postgres dbname=pfe sslmode=disable",
	)
	if err != nil {
		panic(err)
	}

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
	}

	http.HandleFunc("/signup", signup)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/me", me)
	http.HandleFunc("/email-exists", emailExists)
	http.HandleFunc("/apply", applyHandler)
	http.HandleFunc("/applications", authRequired("admin", listApplications))
	http.HandleFunc("/subjects", subjectsHandler)
	http.HandleFunc("/subjects/delete", deleteSubjects)
	http.HandleFunc("/weekly-applications", authRequired("admin", weeklyApplications))
	http.HandleFunc("/uploads/", serveFile)

	fmt.Println("API running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func weeklyApplications(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	var count int
	db.QueryRow(`
		SELECT COUNT(*) FROM applications
		WHERE created_at >= DATE_TRUNC('week', NOW())
	`).Scan(&count)

	json.NewEncoder(w).Encode(map[string]int{
		"count": count,
	})
}

/* ================= AUTH ================= */

func signup(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		return
	}

	var body struct {
		Username string
		Email    string
		Password string
		Role     string
	}

	json.NewDecoder(r.Body).Decode(&body)

	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), 10)

	_, err := db.Exec(`
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1,$2,$3,$4)
	`, body.Username, body.Email, string(hash), body.Role)

	if err != nil {
		http.Error(w, "User exists", 409)
		return
	}

	w.WriteHeader(201)
}

func login(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		return
	}

	var body struct {
		Username string
		Password string
	}

	json.NewDecoder(r.Body).Decode(&body)

	var id int
	var hash, role, username string
	err := db.QueryRow(`
		SELECT id, password_hash, role, username FROM users WHERE username=$1
	`, body.Username).Scan(&id, &hash, &role, &username)

	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		http.Error(w, "Invalid credentials", 401)
		return
	}

	session, _ := store.Get(r, "auth")
	session.Values["user_id"] = id
	session.Values["role"] = role
	session.Values["username"] = username
	session.Save(r, w)
}

func logout(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	session, _ := store.Get(r, "auth")
	session.Options.MaxAge = -1
	session.Save(r, w)
}

func me(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	session, _ := store.Get(r, "auth")

	role, ok := session.Values["role"].(string)
	if !ok {
		json.NewEncoder(w).Encode(map[string]any{"loggedIn": false})
		return
	}

	username, _ := session.Values["username"].(string)

	json.NewEncoder(w).Encode(map[string]any{
		"loggedIn": true,
		"role":     role,
		"username": username,
	})
}

/* ================= APPLICATION ================= */

func applyHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	r.ParseMultipartForm(20 << 20)

	email := r.FormValue("email")

	var exists bool
	db.QueryRow(`SELECT EXISTS (SELECT 1 FROM applications WHERE email=$1)`, email).Scan(&exists)
	if exists {
		http.Error(w, "Email already used", 409)
		return
	}

	savePDF := func(field string) (string, error) {
		file, header, err := r.FormFile(field)
		if err != nil {
			return "", err
		}
		defer file.Close()

		os.MkdirAll("uploads", os.ModePerm)
		path := fmt.Sprintf("uploads/%d_%s", time.Now().UnixNano(), header.Filename)

		dst, _ := os.Create(path)
		defer dst.Close()
		io.Copy(dst, file)
		return path, nil
	}

	cvPath, _ := savePDF("cv")
	motivationPath, _ := savePDF("motivation")

	var startDate sql.NullTime
	if v := r.FormValue("early_start_date"); v != "" {
		t, _ := time.Parse("2006-01-02", v)
		startDate = sql.NullTime{Time: t, Valid: true}
	}

	var appID int
	err := db.QueryRow(`
		INSERT INTO applications (
			full_name, gender, email, phone, university,
			field_of_study, degree_level, application_type,
			internship_duration, preferred_working_method,
			start_date, cv_file_path, motivation_file_path
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id`,
		r.FormValue("full_name"),
		r.FormValue("gender"),
		email,
		r.FormValue("phone"),
		r.FormValue("university"),
		r.FormValue("field_of_study"),
		r.FormValue("degree_level"),
		r.FormValue("application_type"),
		r.FormValue("internship_duration"),
		r.FormValue("preferred_working_method"),
		startDate,
		cvPath,
		motivationPath,
	).Scan(&appID)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	for _, subjectName := range r.Form["subjects"] {
		var subjectID int
		err := db.QueryRow(`SELECT id FROM subjects WHERE name=$1`, subjectName).Scan(&subjectID)
		if err == nil {
			db.Exec(`INSERT INTO application_subjects VALUES ($1,$2)`, appID, subjectID)
		}
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"id":      appID,
	})
}

/* ================= APPLICATION LIST ================= */

func listApplications(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	rows, _ := db.Query(`
		SELECT id, full_name, email, gender, phone, university,
		field_of_study, degree_level, application_type,
		internship_duration, preferred_working_method,
		start_date, created_at, cv_file_path, motivation_file_path
		FROM applications ORDER BY created_at DESC
	`)

	var result []ApplicationResponse

	for rows.Next() {
		var a ApplicationResponse
		var start sql.NullTime
		var created time.Time

		rows.Scan(
			&a.ID, &a.FullName, &a.Email, &a.Gender, &a.Phone,
			&a.University, &a.FieldOfStudy, &a.DegreeLevel,
			&a.ApplicationType, &a.InternshipDuration,
			&a.PreferredWorkingMethod, &start,
			&created, &a.CVFilePath, &a.MotivationFilePath,
		)

		a.CreatedAt = created.Format("2006-01-02")
		if start.Valid {
			s := start.Time.Format("2006-01-02")
			a.StartDate = &s
		}

		subRows, _ := db.Query(`
			SELECT s.name FROM subjects s
			JOIN application_subjects a ON a.subject_id=s.id
			WHERE a.application_id=$1`, a.ID)

		for subRows.Next() {
			var name string
			subRows.Scan(&name)
			a.Subjects = append(a.Subjects, name)
		}

		result = append(result, a)
	}

	json.NewEncoder(w).Encode(result)
}

/* ================= SUBJECTS ================= */

func subjectsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {

	case http.MethodGet:
		rows, _ := db.Query(`SELECT id, name FROM subjects ORDER BY name`)
		defer rows.Close()

		var subjects []map[string]any
		for rows.Next() {
			var id int
			var name string
			rows.Scan(&id, &name)
			subjects = append(subjects, map[string]any{
				"id":   id,
				"name": name,
			})
		}
		json.NewEncoder(w).Encode(subjects)

	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		if body.Name == "" {
			http.Error(w, "Subject name required", 400)
			return
		}

		_, err := db.Exec(`INSERT INTO subjects (name) VALUES ($1)`, body.Name)
		if err != nil {
			http.Error(w, "Subject already exists", 409)
			return
		}

		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	case http.MethodPut:
		var body struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		if body.ID == 0 || body.Name == "" {
			http.Error(w, "Invalid payload", 400)
			return
		}

		_, err := db.Exec(`UPDATE subjects SET name=$1 WHERE id=$2`, body.Name, body.ID)
		if err != nil {
			http.Error(w, "Subject name already exists", 409)
			return
		}

		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	default:
		http.Error(w, "Method not allowed", 405)
	}
}

/* ================= FILE SERVING ================= */

func serveFile(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Serve the file
	http.ServeFile(w, r, r.URL.Path[1:])
}

/* ================= EMAIL ================= */

func emailExists(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	email := r.URL.Query().Get("email")

	var exists bool
	db.QueryRow(`SELECT EXISTS (SELECT 1 FROM applications WHERE email=$1)`, email).Scan(&exists)

	json.NewEncoder(w).Encode(map[string]bool{"exists": exists})
}
