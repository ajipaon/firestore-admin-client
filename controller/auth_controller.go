package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"firestore-admin-client/config"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AuthController struct {
	Config config.Config
}

func NewAuthController(cfg config.Config) *AuthController {
	return &AuthController{
		Config: cfg,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// ServeLoginPage serves the login page
// @Summary Login Page
// @Description Serves the HTML login page
// @Tags auth
// @Produce html
// @Success 200 {string} string "HTML login page"
// @Router /login [get]
func (c *AuthController) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/login.html")
}

// HandleLogin handles Firebase authentication (HTML response)
// @Summary Login (Form)
// @Description Login with email and password using HTML form
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce html
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Success 200 {string} string "HTML response with token"
// @Failure 400 {string} string "HTML error response"
// @Failure 401 {string} string "HTML error response"
// @Router /auth/login [post]
func (c *AuthController) HandleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Failed to parse form data</div>`))
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Email and password are required</div>`))
		return
	}

	ctx := context.Background()
	token, err := c.SignInWithEmailPassword(ctx, email, password)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<div class="error">Authentication failed: ` + err.Error() + `</div>`))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<div class="success">Authentication successful!</div>
		<div>
			<strong>Your token:</strong>
			<pre style="word-wrap: break-word; white-space: pre-wrap;">` + token + `</pre>
		</div>
	`))
}

// HandleJSONLogin handles Firebase authentication with JSON request/response
// @Summary Login (JSON API)
// @Description Login with email and password (JSON API)
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} GenericResponse
// @Failure 400 {object} GenericResponse
// @Failure 401 {object} GenericResponse
// @Router /auth/login/json [post]
func (c *AuthController) HandleJSONLogin(w http.ResponseWriter, r *http.Request) {
	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if loginReq.Email == "" || loginReq.Password == "" {
		RespondWithError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	ctx := context.Background()
	token, err := c.SignInWithEmailPassword(ctx, loginReq.Email, loginReq.Password)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Authentication failed: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Message: "Authentication successful",
		Data: map[string]string{
			"token": token,
		},
	})
}

// HandleJSONRegister handles user registration with JSON request/response
// @Summary Register new user (JSON API)
// @Description Register a new user with email and password (JSON API)
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body RegisterRequest true "Registration credentials"
// @Success 201 {object} GenericResponse
// @Failure 400 {object} GenericResponse
// @Router /auth/register/json [post]
func (c *AuthController) HandleJSONRegister(w http.ResponseWriter, r *http.Request) {
	var registerReq RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if registerReq.Email == "" || registerReq.Password == "" {
		RespondWithError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	ctx := context.Background()
	token, err := c.SignUpWithEmailPassword(ctx, registerReq.Email, registerReq.Password)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Registration failed: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusCreated, GenericResponse{
		Success: true,
		Message: "Registration successful",
		Data: map[string]string{
			"token": token,
		},
	})
}

func (c *AuthController) SignInWithEmailPassword(ctx context.Context, email, password string) (string, error) {
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", c.Config.FirebaseApiKey)

	payload := map[string]interface{}{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling request payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request to Firebase Auth: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authentication failed: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing response: %v", err)
	}

	idToken, ok := result["idToken"].(string)
	if !ok {
		return "", fmt.Errorf("idToken not found in response")
	}

	return idToken, nil
}

func (c *AuthController) SignUpWithEmailPassword(ctx context.Context, email, password string) (string, error) {
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signUp?key=%s", c.Config.FirebaseApiKey)

	payload := map[string]interface{}{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling request payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request to Firebase Auth: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registration failed: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing response: %v", err)
	}

	idToken, ok := result["idToken"].(string)
	if !ok {
		return "", fmt.Errorf("idToken not found in response")
	}

	return idToken, nil
}
