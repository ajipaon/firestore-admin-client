package controller

import (
	"context"
	"encoding/json"
	"firestore-admin-client/config"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
)

type DocumentController struct {
	Client *firestore.Client
	Auth   *auth.Client
	Config config.Config
}

func NewDocumentController(client *firestore.Client, authClient *auth.Client, cfg config.Config) *DocumentController {
	return &DocumentController{
		Client: client,
		Auth:   authClient,
		Config: cfg,
	}
}

func (c *DocumentController) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
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
		authToken, err := c.Auth.VerifyIDToken(ctx, token)
		if err != nil {
			RespondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx = context.WithValue(ctx, "userId", authToken.UID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetDocumentHandler reads document from Firestore
// @Summary Get a document
// @Description Retrieve a specific document from a Firestore collection
// @Tags documents
// @Produce json
// @Param collection path string true "Collection name"
// @Param id path string true "Document ID"
// @Success 200 {object} GenericResponse
// @Failure 401 {object} GenericResponse "Unauthorized"
// @Failure 404 {object} GenericResponse "Document not found"
// @Failure 500 {object} GenericResponse "Server error"
// @Security BearerAuth
// @Router /api/collections/{collection}/documents/{id} [get]
func (c *DocumentController) GetDocumentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collection := vars["collection"]
	docID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	docRef := c.Client.Collection(collection).Doc(docID)
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

// QueryDocumentsHandler queries documents from Firestore collection
// @Summary Query documents
// @Description Retrieve documents from a Firestore collection with optional filtering
// @Tags documents
// @Produce json
// @Param collection path string true "Collection name"
// @Param limit query int false "Maximum number of documents to return" default(50)
// @Success 200 {object} GenericResponse
// @Failure 401 {object} GenericResponse "Unauthorized"
// @Failure 500 {object} GenericResponse "Server error"
// @Security BearerAuth
// @Router /api/collections/{collection}/documents [get]
func (c *DocumentController) QueryDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collection := vars["collection"]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	query := r.URL.Query()
	limit := 50
	if query.Get("limit") != "" {
		fmt.Sscanf(query.Get("limit"), "%d", &limit)
	}

	iter := c.Client.Collection(collection).Limit(limit).Documents(ctx)
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

// CreateDocumentHandler creates document in Firestore
// @Summary Create a document
// @Description Create a new document in a Firestore collection
// @Tags documents
// @Accept json
// @Produce json
// @Param collection path string true "Collection name"
// @Param document body map[string]interface{} true "Document data"
// @Success 201 {object} GenericResponse
// @Failure 400 {object} GenericResponse "Bad request"
// @Failure 401 {object} GenericResponse "Unauthorized"
// @Failure 500 {object} GenericResponse "Server error"
// @Security BearerAuth
// @Router /api/collections/{collection}/documents [post]
func (c *DocumentController) CreateDocumentHandler(w http.ResponseWriter, r *http.Request) {
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

	docRef, _, err := c.Client.Collection(collection).Add(ctx, data)
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

// UpdateDocumentHandler updates document in Firestore
// @Summary Update a document
// @Description Update an existing document in a Firestore collection
// @Tags documents
// @Accept json
// @Produce json
// @Param collection path string true "Collection name"
// @Param id path string true "Document ID"
// @Param document body map[string]interface{} true "Document data"
// @Success 200 {object} GenericResponse
// @Failure 400 {object} GenericResponse "Bad request"
// @Failure 401 {object} GenericResponse "Unauthorized"
// @Failure 500 {object} GenericResponse "Server error"
// @Security BearerAuth
// @Router /api/collections/{collection}/documents/{id} [put]
func (c *DocumentController) UpdateDocumentHandler(w http.ResponseWriter, r *http.Request) {
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

	_, err := c.Client.Collection(collection).Doc(docID).Set(ctx, data, firestore.MergeAll)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error updating document: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Message: "Document updated successfully",
	})
}

// DeleteDocumentHandler deletes document from Firestore
// @Summary Delete a document
// @Description Delete a document from a Firestore collection
// @Tags documents
// @Produce json
// @Param collection path string true "Collection name"
// @Param id path string true "Document ID"
// @Success 200 {object} GenericResponse
// @Failure 401 {object} GenericResponse "Unauthorized"
// @Failure 500 {object} GenericResponse "Server error"
// @Security BearerAuth
// @Router /api/collections/{collection}/documents/{id} [delete]
func (c *DocumentController) DeleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collection := vars["collection"]
	docID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := c.Client.Collection(collection).Doc(docID).Delete(ctx)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error deleting document: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Message: "Document deleted successfully",
	})
}

// ListCollectionsHandler lists all collections in Firestore
// @Summary List collections
// @Description List all collections in Firestore
// @Tags collections
// @Produce json
// @Success 200 {object} GenericResponse
// @Failure 401 {object} GenericResponse "Unauthorized"
// @Failure 500 {object} GenericResponse "Server error"
// @Security BearerAuth
// @Router /api/collections [get]
func (c *DocumentController) ListCollectionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	iter := c.Client.Collections(ctx)
	var collections []map[string]interface{}

	for {
		collection, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error iterating collections: %v", err))
			return
		}

		docIter := collection.Limit(1000).Documents(ctx)
		docCount := 0
		for {
			_, err := docIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fmt.Printf("Error counting documents: %v\n", err)
				break
			}
			docCount++
		}

		collections = append(collections, map[string]interface{}{
			"id":            collection.ID,
			"path":          collection.Path,
			"documentCount": docCount,
		})
	}

	RespondWithJSON(w, http.StatusOK, GenericResponse{
		Success: true,
		Data:    collections,
	})
}
