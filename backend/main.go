// @title Internship Application API
// @version 1.0
// @description API for managing internship applications, subjects, and authentication
// @termsOfService http://example.com/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey SessionAuth
// @in cookie
// @name auth

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "backend/docs"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/gorilla/sessions"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var store = sessions.NewCookieStore([]byte("super-secret-key"))

// ApplicationResponse represents an internship application
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

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "PUT, GET, POST, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func respondError(w http.ResponseWriter, message string, code int) {
	http.Error(w, message, code)
}

func respondJSON(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

func authRequired(role string, next http.HandlerFunc) http.HandlerFunc {
	return corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "auth")
		if err != nil {
			respondError(w, "Session error", http.StatusInternalServerError)
			return
		}

		userRole, ok := session.Values["role"].(string)
		if !ok {
			respondError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if role != "" && userRole != role {
			respondError(w, "Forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

func main() {
	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Database connection attempt %d failed: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	defer db.Close()

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	http.HandleFunc("/signup", corsMiddleware(signup))
	http.HandleFunc("/login", corsMiddleware(login))
	http.HandleFunc("/logout", corsMiddleware(logout))
	http.HandleFunc("/me", corsMiddleware(me))
	http.HandleFunc("/email-exists", corsMiddleware(emailExists))
	http.HandleFunc("/apply", corsMiddleware(applyHandler))
	http.HandleFunc("/subjects", corsMiddleware(subjectsHandler))
	http.HandleFunc("/applications", authRequired("admin", listApplications))
	http.HandleFunc("/subjects/delete", authRequired("admin", deleteSubjects))
	http.HandleFunc("/weekly-applications", authRequired("admin", weeklyApplications))
	http.HandleFunc("/uploads/", corsMiddleware(serveFile))
	http.Handle("/swagger/", httpSwagger.WrapHandler)

	log.Println("API running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func weeklyApplications(w http.ResponseWriter, r *http.Request) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM applications
		WHERE created_at >= DATE_TRUNC('week', NOW())
	`).Scan(&count)

	if err != nil {
		log.Printf("Error getting weekly applications: %v", err)
		respondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]int{"count": count}, http.StatusOK)
}

// signup godoc
// @Summary Create a new user
// @Description Register a new user account
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body object{username=string,email=string,password=string,role=string} true "User payload"
// @Success 201
// @Failure 400 {string} string
// @Failure 409 {string} string
// @Router /signup [post]

func signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Username == "" || body.Email == "" || body.Password == "" {
		respondError(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		respondError(w, "Server error", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
	`, body.Username, body.Email, string(hash), body.Role)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			respondError(w, "User already exists", http.StatusConflict)
			return
		}
		log.Printf("Error creating user: %v", err)
		respondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// login godoc
// @Summary Login
// @Description Authenticate user and create session
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body object{username=string,password=string} true "Login payload"
// @Success 200 {object} map[string]bool
// @Failure 401 {string} string
// @Router /login [post]
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Username == "" || body.Password == "" {
		respondError(w, "Missing credentials", http.StatusBadRequest)
		return
	}

	var id int
	var hash, role, username string
	err := db.QueryRow(`
		SELECT id, password_hash, role, username FROM users WHERE username=$1
	`, body.Username).Scan(&id, &hash, &role, &username)

	if err == sql.ErrNoRows {
		respondError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Error fetching user: %v", err)
		respondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)); err != nil {
		respondError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	session, _ := store.Get(r, "auth")
	session.Values["user_id"] = id
	session.Values["role"] = role
	session.Values["username"] = username

	if err := session.Save(r, w); err != nil {
		log.Printf("Error saving session: %v", err)
		respondError(w, "Session error", http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]bool{"success": true}, http.StatusOK)
}

func logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, _ := store.Get(r, "auth")
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		log.Printf("Error clearing session: %v", err)
		respondError(w, "Session error", http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]bool{"success": true}, http.StatusOK)
}

// me godoc
// @Summary Current user info
// @Description Returns current logged-in user
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /me [get]
func me(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth")
	if err != nil {
		respondJSON(w, map[string]bool{"loggedIn": false}, http.StatusOK)
		return
	}

	role, ok := session.Values["role"].(string)
	if !ok {
		respondJSON(w, map[string]bool{"loggedIn": false}, http.StatusOK)
		return
	}

	username, _ := session.Values["username"].(string)
	respondJSON(w, map[string]interface{}{
		"loggedIn": true,
		"role":     role,
		"username": username,
	}, http.StatusOK)
}

// applyHandler godoc
// @Summary Submit application
// @Description Submit internship application with files
// @Tags Applications
// @Accept multipart/form-data
// @Produce json
// @Param full_name formData string true "Full name"
// @Param email formData string true "Email"
// @Param cv formData file true "CV PDF"
// @Param motivation formData file false "Motivation letter"
// @Param subjects formData []string false "Subjects"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {string} string
// @Router /apply [post]
func applyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		respondError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		respondError(w, "Email is required", http.StatusBadRequest)
		return
	}

	var exists bool
	err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM applications WHERE email=$1)`, email).Scan(&exists)
	if err != nil {
		log.Printf("Error checking email: %v", err)
		respondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	if exists {
		respondError(w, "Email already used", http.StatusConflict)
		return
	}

	savePDF := func(field string) (string, error) {
		file, header, err := r.FormFile(field)
		if err != nil {
			return "", err
		}
		defer file.Close()

		if err := os.MkdirAll("uploads", 0755); err != nil {
			return "", err
		}

		path := fmt.Sprintf("uploads/%d_%s", time.Now().UnixNano(), header.Filename)
		dst, err := os.Create(path)
		if err != nil {
			return "", err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			return "", err
		}
		return path, nil
	}

	cvPath, err := savePDF("cv")
	if err != nil {
		log.Printf("Error saving CV: %v", err)
		respondError(w, "Failed to save CV", http.StatusInternalServerError)
		return
	}

	motivationPath, _ := savePDF("motivation")

	var startDate sql.NullTime
	if v := r.FormValue("early_start_date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			startDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	var appID int
	err = db.QueryRow(`
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
		log.Printf("Error creating application: %v", err)
		respondError(w, "Failed to create application", http.StatusInternalServerError)
		return
	}

	for _, subjectName := range r.Form["subjects"] {
		var subjectID int
		err := db.QueryRow(`SELECT id FROM subjects WHERE name=$1`, subjectName).Scan(&subjectID)
		if err == nil {
			db.Exec(`INSERT INTO application_subjects VALUES ($1,$2)`, appID, subjectID)
		}
	}

	respondJSON(w, map[string]interface{}{
		"success": true,
		"id":      appID,
	}, http.StatusCreated)
}

// listApplications godoc
// @Summary List applications
// @Description Admin: list all applications
// @Tags Admin
// @Produce json
// @Security SessionAuth
// @Success 200 {array} ApplicationResponse
// @Failure 403 {string} string
// @Router /applications [get]
func listApplications(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, full_name, email, gender, phone, university,
		field_of_study, degree_level, application_type,
		internship_duration, preferred_working_method,
		start_date, created_at, cv_file_path, motivation_file_path
		FROM applications ORDER BY created_at DESC
	`)
	if err != nil {
		log.Printf("Error fetching applications: %v", err)
		respondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result []ApplicationResponse

	for rows.Next() {
		var a ApplicationResponse
		var start sql.NullTime
		var created time.Time

		if err := rows.Scan(
			&a.ID, &a.FullName, &a.Email, &a.Gender, &a.Phone,
			&a.University, &a.FieldOfStudy, &a.DegreeLevel,
			&a.ApplicationType, &a.InternshipDuration,
			&a.PreferredWorkingMethod, &start,
			&created, &a.CVFilePath, &a.MotivationFilePath,
		); err != nil {
			log.Printf("Error scanning application: %v", err)
			continue
		}

		a.CreatedAt = created.Format("2006-01-02")
		if start.Valid {
			s := start.Time.Format("2006-01-02")
			a.StartDate = &s
		}

		subRows, err := db.Query(`
			SELECT s.name FROM subjects s
			JOIN application_subjects a ON a.subject_id=s.id
			WHERE a.application_id=$1`, a.ID)

		if err == nil {
			defer subRows.Close()
			for subRows.Next() {
				var name string
				if err := subRows.Scan(&name); err == nil {
					a.Subjects = append(a.Subjects, name)
				}
			}
		}
		result = append(result, a)
	}

	respondJSON(w, result, http.StatusOK)
}

// subjectsHandler godoc
// @Summary Manage subjects
// @Description Get, create, or update subjects
// @Tags Subjects
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /subjects [get]
func subjectsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(`SELECT id, name FROM subjects ORDER BY name`)
		if err != nil {
			log.Printf("Error fetching subjects: %v", err)
			respondError(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var subjects []map[string]interface{}
		for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				continue
			}
			subjects = append(subjects, map[string]interface{}{
				"id":   id,
				"name": name,
			})
		}
		respondJSON(w, subjects, http.StatusOK)

	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respondError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if body.Name == "" {
			respondError(w, "Subject name required", http.StatusBadRequest)
			return
		}

		_, err := db.Exec(`INSERT INTO subjects (name) VALUES ($1)`, body.Name)
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				respondError(w, "Subject already exists", http.StatusConflict)
				return
			}
			log.Printf("Error creating subject: %v", err)
			respondError(w, "Database error", http.StatusInternalServerError)
			return
		}
		respondJSON(w, map[string]bool{"success": true}, http.StatusCreated)

	case http.MethodPut:
		var body struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respondError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if body.ID == 0 || body.Name == "" {
			respondError(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		_, err := db.Exec(`UPDATE subjects SET name=$1 WHERE id=$2`, body.Name, body.ID)
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				respondError(w, "Subject name already exists", http.StatusConflict)
				return
			}
			log.Printf("Error updating subject: %v", err)
			respondError(w, "Database error", http.StatusInternalServerError)
			return
		}
		respondJSON(w, map[string]bool{"success": true}, http.StatusOK)

	default:
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func deleteSubjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		IDs []int `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(payload.IDs) == 0 {
		respondError(w, "No subjects selected", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
			SELECT DISTINCT subject_id
			FROM application_subjects
			WHERE subject_id = ANY($1)
		`, pq.Array(payload.IDs))
	if err != nil {
		log.Printf("Error checking subject usage: %v", err)
		respondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	inUse := map[int]bool{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			inUse[id] = true
		}
	}
	var deletable []int
	for _, id := range payload.IDs {
		if !inUse[id] {
			deletable = append(deletable, id)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		respondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if len(deletable) > 0 {
		if _, err := tx.Exec(`
			DELETE FROM subjects WHERE id = ANY($1)
		`, pq.Array(deletable)); err != nil {
			log.Printf("Error deleting subjects: %v", err)
			respondError(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		respondError(w, "Database error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"deleted": deletable,
		"in_use":  mapKeys(inUse),
	}, http.StatusOK)
}

func serveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := r.URL.Path[1:]
	if filePath == "" || filePath[0] == '.' {
		respondError(w, "Invalid file path", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, filePath)
}

func emailExists(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		respondError(w, "Email parameter required", http.StatusBadRequest)
		return
	}

	var exists bool
	err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM applications WHERE email=$1)`, email).Scan(&exists)
	if err != nil {
		log.Printf("Error checking email: %v", err)
		respondError(w, "Database error", http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]bool{"exists": exists}, http.StatusOK)
}

func mapKeys(m map[int]bool) []int {
	var keys []int
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
