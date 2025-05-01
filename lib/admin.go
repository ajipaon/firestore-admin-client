package lib

import (
	"context"
	"encoding/base64"
	"firestore-admin-client/config"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

type AdminHandler struct {
	Config  config.Config
	Store   *sessions.CookieStore
	Handler *FirestoreHandler
}

const (
	SessionName = "admin-session"
	SessionKey  = "authenticated"
)

func NewAdminHandler(cfg config.Config, firestoreHandler *FirestoreHandler) *AdminHandler {
	store := sessions.NewCookieStore([]byte(uuid.New().String()))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 8, // 8 hours
		HttpOnly: true,
	}

	return &AdminHandler{
		Config:  cfg,
		Store:   store,
		Handler: firestoreHandler,
	}
}

func (a *AdminHandler) ServeAdminLoginPage(w http.ResponseWriter, r *http.Request) {
	// Check if already authenticated
	session, _ := a.Store.Get(r, SessionName)
	if auth, ok := session.Values[SessionKey].(bool); ok && auth {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	http.ServeFile(w, r, "static/admin-login.html")
}

func (a *AdminHandler) HandleAdminLogin(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<div class="error">Invalid username or password</div>`))
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

func (a *AdminHandler) HandleAdminLogout(w http.ResponseWriter, r *http.Request) {
	// Clear session
	session, _ := a.Store.Get(r, SessionName)
	session.Values[SessionKey] = false
	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *AdminHandler) ServeAdminDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/admin-dashboard.html")
}

func (a *AdminHandler) AdminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check session
		session, _ := a.Store.Get(r, SessionName)
		if auth, ok := session.Values[SessionKey].(bool); !ok || !auth {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (a *AdminHandler) BasicAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
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

func (a *AdminHandler) GetUserCount(w http.ResponseWriter, r *http.Request) {
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

func (a *AdminHandler) getUserCount(ctx context.Context) (int, error) {
	iter := a.Handler.Auth.Users(ctx, "")
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
