package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"api/models"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// Env for persisting states global to server
type Env struct {
	db models.Datastore
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Heartbeat("/"))
	r.Use(middleware.Recoverer)

	db, err := models.NewDB()
	if err != nil {
		log.Panic(err)
	}
	env := &Env{db}

	r.Get("/stats", env.getStats)

	r.Get("/projects", env.getProjects)
	r.Get("/projects/{id}", env.getProject)
	r.Get("/projects/{id}/comments", env.getProjectComments)
	r.Put("/projects/{id}", env.putProject)
	r.Post("/project", env.createProject)

	r.Post("/auth", env.loginUser)
	r.Post("/user", env.createUser)
	r.Get("/users", env.getUsers)

	r.Get("/payments", env.getPayments)
	r.Post("/payments", env.createPayment)
	r.Delete("/payments/{id}", env.deletePayment)

	r.Get("/categories", env.getCategories)
	r.Post("/category", env.createCategory)

	r.Get("/comments", env.getComments)
	r.Post("/comments", env.createComment)
	r.Delete("/comments/{id}", env.deleteComment)

	port := os.Getenv("PORT")
	log.Println("Running server at port " + port)
	http.ListenAndServe(":"+port, r)
}

func secretKey() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

func (env *Env) getStats(w http.ResponseWriter, r *http.Request) {
	stats, err := env.db.AllStatistics()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, stats)
}

func (env *Env) getProjects(w http.ResponseWriter, r *http.Request) {
	numPerPage, pageIndex := getPaginationCursor(r)

	var projects interface{}
	var err error

	searchTerm := r.URL.Query().Get("search")
	if searchTerm == "" {
		projects, err = env.db.AllProjects(numPerPage, pageIndex)
	} else {
		projects, err = env.db.SearchProjects(searchTerm, numPerPage, pageIndex)
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, projects)
}

func (env *Env) getProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "No such project id")
		return
	}
	project, err := env.db.GetProject(projectID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, project)
}

func (env *Env) getProjectComments(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "No such project id")
		return
	}
	numPerPage, pageIndex := getPaginationCursor(r)
	comments, err := env.db.AllProjectComments(projectID, numPerPage, pageIndex)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, comments)
}

type projectRequest struct {
	Title          string    `json:"title"`
	UserID         int       `json:"userId"`
	Category       string    `json:"category"`
	Description    string    `json:"description"`
	Image          string    `json:"image"`
	Verified       bool      `json:"verified,omitempty"`
	AmountRequired float64   `json:"amountRequired"`
	EndTime        time.Time `json:"endTime"`
}

func (env *Env) putProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "No such project id")
		return
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var project projectRequest
	err = decoder.Decode(&project)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	err = env.db.ReplaceProject(
		projectID,
		project.Title,
		project.UserID,
		project.Category,
		project.Description,
		project.Image,
		project.Verified,
		project.AmountRequired,
		project.EndTime,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusNoContent, projectID)
}

func (env *Env) createProject(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var project projectRequest
	err := decoder.Decode(&project)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	projectID, err := env.db.CreateProject(
		project.Title,
		project.UserID,
		project.Category,
		project.Description,
		project.Image,
		project.AmountRequired,
		project.EndTime,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, projectID)
}

func (env *Env) getUsers(w http.ResponseWriter, r *http.Request) {
	numPerPage, pageIndex := getPaginationCursor(r)
	users, err := env.db.AllUsers(numPerPage, pageIndex)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, users)
}

func userAuth(w http.ResponseWriter, user *models.User) {
	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		ExpiresAt: 15000,
		Issuer:    "Givingchy",
	})
	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(secretKey())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := map[string]interface{}{
		"userId":   user.ID,
		"username": user.Username,
		"image":    user.Image,
		"isAdmin":  user.IsAdmin,
		"token":    tokenString,
	}
	respondWithJSON(w, http.StatusCreated, data)
}

type userRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (env *Env) loginUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var userReq userRequest
	err := decoder.Decode(&userReq)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := env.db.GetUser(userReq.Email, userReq.Password)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	userAuth(w, user)
}

type newUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
	Image    string `json:"image"`
}

func (env *Env) createUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var u newUserRequest
	err := decoder.Decode(&u)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := env.db.CreateUser(u.Email, u.Password, u.Username, u.Image)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	userAuth(w, user)
}

func (env *Env) getCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := env.db.AllCategories()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, categories)
}

func (env *Env) getPayments(w http.ResponseWriter, r *http.Request) {
	numPerPage, pageIndex := getPaginationCursor(r)
	users, err := env.db.AllPayments(numPerPage, pageIndex)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, users)
}

type paymentRequest struct {
	UserID    int     `json:"userId"`
	ProjectID int     `json:"projectId"`
	Amount    float64 `json:"amount"`
}

func (env *Env) createPayment(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payment paymentRequest
	err := decoder.Decode(&payment)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	paymentID, err := env.db.CreatePayment(
		payment.UserID,
		payment.ProjectID,
		payment.Amount,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, paymentID)
}

func (env *Env) deletePayment(w http.ResponseWriter, r *http.Request) {
	paymentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "No such payment id")
		return
	}
	if err := env.db.DeletePayment(paymentID); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (env *Env) createCategory(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var category models.Category
	err := decoder.Decode(&category)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	err = env.db.CreateCategory(category.Name)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, category)
}

func (env *Env) getComments(w http.ResponseWriter, r *http.Request) {
	numPerPage, pageIndex := getPaginationCursor(r)
	comments, err := env.db.AllComments(numPerPage, pageIndex)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, comments)
}

type commentRequest struct {
	UserID    int    `json:"userId"`
	ProjectID int    `json:"projectId"`
	Content   string `json:"content"`
}

func (env *Env) createComment(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var comment commentRequest
	err := decoder.Decode(&comment)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	commentID, err := env.db.CreateComment(
		comment.UserID,
		comment.ProjectID,
		comment.Content,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, commentID)
}

func (env *Env) deleteComment(w http.ResponseWriter, r *http.Request) {
	commentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "No such comment id")
		return
	}
	if err := env.db.DeleteComment(commentID); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func getPaginationCursor(r *http.Request) (numPerPage, pageIndex int) {
	// look at URL query params for page number and limit
	queryParams := r.URL.Query()
	pageNum, err := strconv.Atoi(queryParams.Get("page"))
	if err != nil {
		pageNum = 1
	}
	limit, err := strconv.Atoi(queryParams.Get("limit"))
	if err != nil {
		limit = 10
	}
	return limit, pageNum
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
