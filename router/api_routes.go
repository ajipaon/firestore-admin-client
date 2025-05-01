package router

import (
	"firestore-admin-client/controller"

	"github.com/gorilla/mux"
)

func SetupAPIRoutes(router *mux.Router, docController *controller.DocumentController) {
	// API endpoints with authentication
	api := router.PathPrefix("/api").Subrouter()

	// Collection operations
	api.HandleFunc("/collections", docController.AuthMiddleware(docController.ListCollectionsHandler)).Methods("GET")

	// Document operations
	api.HandleFunc("/collections/{collection}/documents", docController.AuthMiddleware(docController.QueryDocumentsHandler)).Methods("GET")
	api.HandleFunc("/collections/{collection}/documents", docController.AuthMiddleware(docController.CreateDocumentHandler)).Methods("POST")
	api.HandleFunc("/collections/{collection}/documents/{id}", docController.AuthMiddleware(docController.GetDocumentHandler)).Methods("GET")
	api.HandleFunc("/collections/{collection}/documents/{id}", docController.AuthMiddleware(docController.UpdateDocumentHandler)).Methods("PUT", "PATCH")
	api.HandleFunc("/collections/{collection}/documents/{id}", docController.AuthMiddleware(docController.DeleteDocumentHandler)).Methods("DELETE")
}
