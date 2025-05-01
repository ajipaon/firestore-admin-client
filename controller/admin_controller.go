package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"firestore-admin-client/config"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/auth"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

type AdminController struct {
	Config config.Config
	Store  *sessions.CookieStore
	Auth   *auth.Client
}

const (
	SessionName = "admin-session"
	SessionKey  = "authenticated"
)

func NewAdminController(cfg config.Config, authClient *auth.Client) *AdminController {

	store := sessions.NewCookieStore([]byte(uuid.New().String()))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 8,
		HttpOnly: true,
	}

	return &AdminController{
		Config: cfg,
		Store:  store,
		Auth:   authClient,
	}
}

func (a *AdminController) ServeAdminLoginPage(w http.ResponseWriter, r *http.Request) {

	session, _ := a.Store.Get(r, SessionName)
	if auth, ok := session.Values[SessionKey].(bool); ok && auth {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	http.ServeFile(w, r, "static/admin-login.html")
}

func (a *AdminController) HandleAdminLogin(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Failed to parse form data</div>`))
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username != a.Config.AdminUsername || password != a.Config.AdminPassword {
		RespondWithJSON(w, http.StatusUnauthorized, GenericResponse{
			Success: false,
			Error:   "Invalid username or password",
		})
		return
	}

	session, _ := a.Store.Get(r, SessionName)
	session.Values[SessionKey] = true
	session.Save(r, w)

	w.Header().Set("HX-Redirect", "/admin/dashboard")
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<div class="success">Login successful! Redirecting...</div>`))
}

func (a *AdminController) HandleAdminLogout(w http.ResponseWriter, r *http.Request) {
	// Clear session
	session, _ := a.Store.Get(r, SessionName)
	session.Values[SessionKey] = false
	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *AdminController) ServeAdminDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/admin-dashboard-new.html")
}

func (a *AdminController) ServeAdminUserPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/admin-user.html")
}

func (a *AdminController) ServeFirestoreCollectionsPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/admin-firestore.html")
}

func (a *AdminController) AdminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := a.Store.Get(r, SessionName)
		if auth, ok := session.Values[SessionKey].(bool); !ok || !auth {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (a *AdminController) BasicAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Admin Area"`)
			RespondWithError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		if !strings.HasPrefix(auth, "Basic ") {
			RespondWithError(w, http.StatusUnauthorized, "Invalid authentication method")
			return
		}

		payload, err := base64.StdEncoding.DecodeString(auth[6:])
		if err != nil {
			RespondWithError(w, http.StatusUnauthorized, "Invalid authentication header")
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 {
			RespondWithError(w, http.StatusUnauthorized, "Invalid authentication header")
			return
		}

		if pair[0] != a.Config.AdminUsername || pair[1] != a.Config.AdminPassword {
			RespondWithError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (a *AdminController) GetUserCount(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userCount, err := a.getUserCount(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf("%d", userCount)))
}

func (a *AdminController) getUserCount(ctx context.Context) (int, error) {

	iter := a.Auth.Users(ctx, "")
	var count int

	for {
		_, err := iter.Next()
		if err != nil {
			break
		}
		count++
	}

	return count, nil
}

type UserData struct {
	UID         string `json:"uid"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	PhoneNumber string `json:"phoneNumber"`
	CreatedAt   string `json:"createdAt"`
	LastSignIn  string `json:"lastSignIn"`
	Disabled    bool   `json:"disabled"`
}

type CreateUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

func (a *AdminController) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithJSON(w, http.StatusBadRequest, GenericResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	if req.Email == "" {
		RespondWithJSON(w, http.StatusBadRequest, GenericResponse{
			Success: false,
			Error:   "Email is required",
		})
		return
	}

	password := req.Password
	isGenerated := false
	if password == "" {
		// Generate a random password
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"
		const length = 12

		rand.Seed(time.Now().UnixNano())
		passwordBytes := make([]byte, length)
		for i := range passwordBytes {
			passwordBytes[i] = charset[rand.Intn(len(charset))]
		}
		password = string(passwordBytes)
		isGenerated = true
	}

	params := (&auth.UserToCreate{}).
		Email(req.Email).
		Password(password).
		EmailVerified(false)

	if req.DisplayName != "" {
		params = params.DisplayName(req.DisplayName)
	}

	user, err := a.Auth.CreateUser(ctx, params)
	if err != nil {
		RespondWithJSON(w, http.StatusInternalServerError, GenericResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create user: %v", err),
		})
		return
	}

	responseData := map[string]interface{}{
		"uid":   user.UID,
		"email": user.Email,
	}

	if isGenerated {
		responseData["password"] = password
		responseData["passwordGenerated"] = true
	}

	RespondWithJSON(w, http.StatusCreated, GenericResponse{
		Success: true,
		Message: "User created successfully",
		Data:    responseData,
	})
}

func (a *AdminController) GetUsers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")

	page := 1
	pageSize := 10

	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
		if page < 1 {
			page = 1
		}
	}

	if pageSizeStr != "" {
		fmt.Sscanf(pageSizeStr, "%d", &pageSize)
		if pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}
	}

	iter := a.Auth.Users(ctx, "")
	var users []UserData
	var count int
	var skipped int

	skip := (page - 1) * pageSize

	for {
		user, err := iter.Next()
		if err != nil {
			break
		}

		count++

		if skipped < skip {
			skipped++
			continue
		}

		if len(users) < pageSize {
			createdAt := ""
			lastSignIn := ""

			if user.UserMetadata.CreationTimestamp > 0 {
				createdAt = time.Unix(user.UserMetadata.CreationTimestamp/1000, 0).Format(time.RFC3339)
			}

			if user.UserMetadata.LastLogInTimestamp > 0 {
				lastSignIn = time.Unix(user.UserMetadata.LastLogInTimestamp/1000, 0).Format(time.RFC3339)
			}

			userData := UserData{
				UID:         user.UID,
				Email:       user.Email,
				DisplayName: user.DisplayName,
				PhoneNumber: user.PhoneNumber,
				CreatedAt:   createdAt,
				LastSignIn:  lastSignIn,
				Disabled:    user.Disabled,
			}
			users = append(users, userData)
		} else {
			continue
		}
	}

	totalPages := (count + pageSize - 1) / pageSize

	response := map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"currentPage": page,
			"pageSize":    pageSize,
			"totalItems":  count,
			"totalPages":  totalPages,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (a *AdminController) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req struct {
		UID string `json:"uid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithJSON(w, http.StatusBadRequest, GenericResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	if req.UID == "" {
		RespondWithJSON(w, http.StatusBadRequest, GenericResponse{
			Success: false,
			Error:   "User ID is required",
		})
		return
	}

	if err := a.Auth.DeleteUser(ctx, req.UID); err != nil {
		RespondWithJSON(w, http.StatusInternalServerError, GenericResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to delete user: %v", err),
		})
		return
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Message: "User deleted successfully",
	})
}

func (a *AdminController) ToggleUserStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req struct {
		UID      string `json:"uid"`
		Disabled bool   `json:"disabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithJSON(w, http.StatusBadRequest, GenericResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	if req.UID == "" {
		RespondWithJSON(w, http.StatusBadRequest, GenericResponse{
			Success: false,
			Error:   "User ID is required",
		})
		return
	}

	params := (&auth.UserToUpdate{}).
		Disabled(req.Disabled)

	_, err := a.Auth.UpdateUser(ctx, req.UID, params)
	if err != nil {
		RespondWithJSON(w, http.StatusInternalServerError, GenericResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to update user status: %v", err),
		})
		return
	}

	status := "disabled"
	if !req.Disabled {
		status = "enabled"
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Message: fmt.Sprintf("User %s successfully", status),
	})
}

func (a *AdminController) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req struct {
		UID string `json:"uid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithJSON(w, http.StatusBadRequest, GenericResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	if req.UID == "" {
		RespondWithJSON(w, http.StatusBadRequest, GenericResponse{
			Success: false,
			Error:   "User ID is required",
		})
		return
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"
	const length = 12

	rand.Seed(time.Now().UnixNano())
	password := make([]byte, length)
	for i := range password {
		password[i] = charset[rand.Intn(len(charset))]
	}
	newPassword := string(password)

	params := (&auth.UserToUpdate{}).
		Password(newPassword)

	_, err := a.Auth.UpdateUser(ctx, req.UID, params)
	if err != nil {
		RespondWithJSON(w, http.StatusInternalServerError, GenericResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to reset password: %v", err),
		})
		return
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Message: "Password reset successfully",
		Data: map[string]interface{}{
			"password": newPassword,
		},
	})
}
