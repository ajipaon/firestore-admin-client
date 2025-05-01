package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/login.html")
}

func (h *FirestoreHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {

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
	token, err := h.SignInWithEmailPassword(ctx, email, password)
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
func (h *FirestoreHandler) HandleJSONLogin(w http.ResponseWriter, r *http.Request) {

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
	token, err := h.SignInWithEmailPassword(ctx, loginReq.Email, loginReq.Password)
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

func (h *FirestoreHandler) SignInWithEmailPassword(ctx context.Context, email, password string) (string, error) {
	// Firebase Auth REST API endpoint for email/password sign-in
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", h.Config.FirebaseApiKey)

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
