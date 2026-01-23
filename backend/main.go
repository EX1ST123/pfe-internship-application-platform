package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	_ "backend/docs"

	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger"
)

var db *sql.DB
var err error

// @title PFE Application API
// @version 1.0
// @description This is a backend API for PFE applications
// @host localhost:8080
// @BasePath /

// @contact.name API Support
// @contact.email support@pfe.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// ApplicationResponse represents an application in API responses
// @Description Detailed information about a PFE application
type ApplicationResponse struct {
	ID                     int     `json:"id" example:"1"`
	FullName               string  `json:"full_name" example:"John Doe"`
	Email                  string  `json:"email" example:"john@example.com"`
	Gender                 string  `json:"gender" example:"Male"`
	Phone                  string  `json:"phone" example:"+1234567890"`
	University             string  `json:"university" example:"University of Technology"`
	FieldOfStudy           string  `json:"field_of_study" example:"Computer Science"`
	DegreeLevel            string  `json:"degree_level" example:"Master"`
	ApplicationType        string  `json:"application_type" example:"Internship"`
	InternshipDuration     string  `json:"internship_duration" example:"6 months"`
	PreferredWorkingMethod string  `json:"preferred_working_method" example:"Remote"`
	StartDate              *string `json:"start_date,omitempty" example:"2024-01-15"`
	CreatedAt              string  `json:"created_at" example:"2024-01-10"`
	CVFilePath             string  `json:"cv_file_path" example:"uploads/123456789_cv.pdf"`
	MotivationFilePath     string  `json:"motivation_file_path,omitempty" example:"uploads/123456789_motivation.pdf"`
}

// SuccessResponse represents a standard success response
// @Description Standard success response structure
type SuccessResponse struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message" example:"Operation completed successfully"`
	ID      int         `json:"id,omitempty" example:"1"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse represents a standard error response
// @Description Standard error response structure
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Message string `json:"message" example:"An error occurred"`
	Error   string `json:"error,omitempty" example:"detailed error description"`
}

// WeeklyStatsResponse represents weekly application statistics
// @Description Weekly application statistics
type WeeklyStatsResponse struct {
	Count     int    `json:"count" example:"15"`
	WeekStart string `json:"week_start" example:"2024-01-22"`
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func main() {
	db, err = sql.Open(
		"postgres",
		"host=localhost port=5432 user=postgres password=postgres dbname=form sslmode=disable",
	)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	// Serve uploaded files
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PFE Backend API is running"))
	})

	http.HandleFunc("/apply", applyHandler)
	http.HandleFunc("/applications", listApplications)
	http.HandleFunc("/weekly-applications", getWeeklyApplications)
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	fmt.Println("Server running on http://localhost:8080")
	fmt.Println("Swagger UI available at http://localhost:8080/swagger/index.html")
	http.ListenAndServe(":8080", nil)
}

// Apply for PFE
// @Summary Submit a new PFE application
// @Description Submit a complete PFE application with personal details and required documents
// @Tags applications
// @Accept mpfd
// @Produce json
// @Param full_name formData string true "Applicant's full name"
// @Param email formData string true "Email address"
// @Param gender formData string true "Gender (Male/Female/Other)"
// @Param phone formData string true "Phone number"
// @Param application_type formData string true "Type of application (Internship/Project)"
// @Param university formData string true "University name"
// @Param field_of_study formData string true "Field of study"
// @Param degree_level formData string true "Degree level (Bachelor/Master/PhD)"
// @Param internship_duration formData string true "Duration of internship (3 months/6 months)"
// @Param preferred_working_method formData string true "Preferred working method (Onsite/Remote/Hybrid)"
// @Param early_start_date formData string false "Preferred start date (YYYY-MM-DD)"
// @Param subjects formData []string true "Subjects of interest (at least one required)" collectionFormat(multi)
// @Param cv formData file true "CV document (PDF only)"
// @Param motivation formData file false "Motivation letter (PDF)"
// @Success 201 {object} SuccessResponse "Application submitted successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /apply [post]
func applyHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(20 << 20)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Extract all form values
	fullName := r.FormValue("full_name")
	email := r.FormValue("email")
	gender := r.FormValue("gender")
	phone := r.FormValue("phone")
	applicationType := r.FormValue("application_type")
	university := r.FormValue("university")
	fieldOfStudy := r.FormValue("field_of_study")
	degreeLevel := r.FormValue("degree_level")
	internshipDuration := r.FormValue("internship_duration")
	preferredWorkingMethod := r.FormValue("preferred_working_method")
	startDate := r.FormValue("early_start_date")

	subjects := r.Form["subjects"]
	if len(subjects) == 0 {
		http.Error(w, "At least one subject is required", 400)
		return
	}

	var parsedStartDate sql.NullTime
	if startDate != "" {
		parsed, err := time.Parse("2006-01-02", startDate)
		if err == nil {
			parsedStartDate = sql.NullTime{Time: parsed, Valid: true}
		}
	}

	savePDF := func(field string) (string, error) {
		file, header, err := r.FormFile(field)
		if err != nil {
			return "", nil
		}
		defer file.Close()

		if header.Header.Get("Content-Type") != "application/pdf" {
			return "", fmt.Errorf("%s must be a PDF file", field)
		}

		os.MkdirAll("uploads", os.ModePerm)

		path := "uploads/" + fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
		dst, err := os.Create(path)
		if err != nil {
			return "", err
		}
		defer dst.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			return "", err
		}
		return path, nil
	}

	cvPath, err := savePDF("cv")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	motivationPath, _ := savePDF("motivation")

	var applicationID int
	err = db.QueryRow(`
        INSERT INTO applications (
            full_name, email, gender, phone, application_type,
            university, field_of_study, degree_level,
            internship_duration, preferred_working_method,
            start_date, cv_file_path, motivation_file_path
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        RETURNING id`,
		fullName, email, gender, phone, applicationType,
		university, fieldOfStudy, degreeLevel,
		internshipDuration, preferredWorkingMethod,
		parsedStartDate, cvPath, motivationPath,
	).Scan(&applicationID)

	if err != nil {
		fmt.Printf("Database insert error: %v\n", err)
		http.Error(w, "Failed to save application: "+err.Error(), 500)
		return
	}

	for _, subject := range subjects {
		_, err = db.Exec(
			`INSERT INTO application_subjects (application_id, subject) VALUES ($1, $2)`,
			applicationID, subject,
		)
		if err != nil {
			fmt.Printf("Subject insert error: %v\n", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: "Application submitted successfully",
		ID:      applicationID,
	})
}

// Get all applications
// @Summary List all submitted applications
// @Description Retrieve a list of all PFE applications in descending order of creation
// @Tags applications
// @Accept json
// @Produce json
// @Success 200 {array} ApplicationResponse "List of applications"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /applications [get]
func listApplications(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	rows, err := db.Query(`
        SELECT 
            id, full_name, email, gender, phone, 
            university, field_of_study, degree_level,
            application_type, internship_duration,
            preferred_working_method, start_date,
            created_at, cv_file_path, motivation_file_path
        FROM applications 
        ORDER BY created_at DESC
    `)
	if err != nil {
		fmt.Printf("Database query error: %v\n", err)
		http.Error(w, "Failed to fetch applications", 500)
		return
	}
	defer rows.Close()

	var apps []ApplicationResponse
	for rows.Next() {
		var app ApplicationResponse
		var startDate sql.NullTime
		var rawCreatedAt time.Time
		err := rows.Scan(
			&app.ID, &app.FullName, &app.Email, &app.Gender, &app.Phone,
			&app.University, &app.FieldOfStudy, &app.DegreeLevel,
			&app.ApplicationType, &app.InternshipDuration,
			&app.PreferredWorkingMethod, &startDate,
			&rawCreatedAt, &app.CVFilePath, &app.MotivationFilePath,
		)
		if err != nil {
			fmt.Printf("Row scan error: %v\n", err)
			continue
		}

		app.CreatedAt = rawCreatedAt.Format("2006-01-02")

		if startDate.Valid {
			formattedDate := startDate.Time.Format("2006-01-02")
			app.StartDate = &formattedDate
		}

		apps = append(apps, app)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Rows error: %v\n", err)
		http.Error(w, "Failed to process applications", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

// Get weekly applications count
// @Summary Get count of applications submitted this week
// @Description Returns the number of applications submitted since Monday of the current week
// @Tags analytics
// @Accept json
// @Produce json
// @Success 200 {object} WeeklyStatsResponse "Weekly applications count"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /weekly-applications [get]
func getWeeklyApplications(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	now := time.Now()

	weekday := now.Weekday()
	var daysSinceMonday int
	if weekday == time.Sunday {
		daysSinceMonday = 6
	} else {
		daysSinceMonday = int(weekday) - 1
	}

	startOfWeek := time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, now.Location())

	query := `
        SELECT COUNT(*) 
        FROM applications 
        WHERE created_at >= $1
    `

	var count int
	err := db.QueryRow(query, startOfWeek).Scan(&count)
	if err != nil {
		fmt.Printf("Weekly count error: %v\n", err)
		http.Error(w, "Failed to get weekly count", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WeeklyStatsResponse{
		Count:     count,
		WeekStart: startOfWeek.Format("2006-01-02"),
	})
}
