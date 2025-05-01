package lib

import (
	"context"
	"encoding/json"
	"firestore-admin-client/config"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type FirestoreHandler struct {
	Client *firestore.Client
	Auth   *auth.Client
	Config config.Config
}

func InitFirestore(config config.Config) (*FirestoreHandler, error) {
	ctx := context.Background()
	opt := option.WithCredentialsFile(config.FirebaseCredFile)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing firestore client: %v", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase auth client: %v", err)
	}

	return &FirestoreHandler{
		Client: client,
		Auth:   authClient,
		Config: config,
	}, nil
}

func (h *FirestoreHandler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			RespondWithError(w, http.StatusUnauthorized, "Authorization token required")
			return
		}

		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		ctx := r.Context()
		authToken, err := h.Auth.VerifyIDToken(ctx, token)
		if err != nil {
			RespondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx = context.WithValue(ctx, "userId", authToken.UID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (h *FirestoreHandler) GetDocumentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collection := vars["collection"]
	docID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	docRef := h.Client.Collection(collection).Doc(docID)
	docSnap, err := docRef.Get(ctx)

	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error getting document: %v", err))
		return
	}

	if !docSnap.Exists() {
		RespondWithError(w, http.StatusNotFound, "Document not found")
		return
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Data:    docSnap.Data(),
	})
}

func (h *FirestoreHandler) QueryDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collection := vars["collection"]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	query := r.URL.Query()
	limit := 50 // Default limit
	if query.Get("limit") != "" {
		fmt.Sscanf(query.Get("limit"), "%d", &limit)
	}

	iter := h.Client.Collection(collection).Limit(limit).Documents(ctx)
	var docs []map[string]interface{}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error iterating documents: %v", err))
			return
		}

		data := doc.Data()
		data["id"] = doc.Ref.ID
		docs = append(docs, data)
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Data:    docs,
	})
}

func (h *FirestoreHandler) CreateDocumentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collection := vars["collection"]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	data["createdAt"] = firestore.ServerTimestamp
	if userID, ok := r.Context().Value("userId").(string); ok {
		data["createdBy"] = userID
	}

	docRef, _, err := h.Client.Collection(collection).Add(ctx, data)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating document: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusCreated, GenericResponse{
		Success: true,
		Message: "Document created successfully",
		Data: map[string]string{
			"id": docRef.ID,
		},
	})
}

func (h *FirestoreHandler) UpdateDocumentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collection := vars["collection"]
	docID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	data["updatedAt"] = firestore.ServerTimestamp
	if userID, ok := r.Context().Value("userId").(string); ok {
		data["updatedBy"] = userID
	}

	_, err := h.Client.Collection(collection).Doc(docID).Set(ctx, data, firestore.MergeAll)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error updating document: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Message: "Document updated successfully",
	})
}

func (h *FirestoreHandler) DeleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collection := vars["collection"]
	docID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := h.Client.Collection(collection).Doc(docID).Delete(ctx)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error deleting document: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Message: "Document deleted successfully",
	})
}
