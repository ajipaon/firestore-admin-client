package router

import (
	"firestore-admin-client/controller"

	"github.com/gorilla/mux"
)

func SetupAdminRoutes(router *mux.Router, adminController *controller.AdminController, docController *controller.DocumentController) {
	// Admin authentication
	router.HandleFunc("/admin/login", adminController.ServeAdminLoginPage).Methods("GET")
	router.HandleFunc("/admin/login", adminController.HandleAdminLogin).Methods("POST")
	router.HandleFunc("/admin/logout", adminController.HandleAdminLogout).Methods("GET")

	// Admin dashboard and API
	router.HandleFunc("/admin/dashboard", adminController.AdminAuthMiddleware(adminController.ServeAdminDashboard)).Methods("GET")
	router.HandleFunc("/admin/user", adminController.AdminAuthMiddleware(adminController.ServeAdminUserPage)).Methods("GET")
	router.HandleFunc("/admin/firestore", adminController.AdminAuthMiddleware(adminController.ServeFirestoreCollectionsPage)).Methods("GET")
	router.HandleFunc("/admin/api/user-count", adminController.AdminAuthMiddleware(adminController.GetUserCount)).Methods("GET")
	router.HandleFunc("/admin/api/users", adminController.AdminAuthMiddleware(adminController.GetUsers)).Methods("GET")
	router.HandleFunc("/admin/api/create-user", adminController.AdminAuthMiddleware(adminController.CreateUser)).Methods("POST")
	router.HandleFunc("/admin/api/delete-user", adminController.AdminAuthMiddleware(adminController.DeleteUser)).Methods("DELETE")
	router.HandleFunc("/admin/api/toggle-user-status", adminController.AdminAuthMiddleware(adminController.ToggleUserStatus)).Methods("PUT")
	router.HandleFunc("/admin/api/reset-password", adminController.AdminAuthMiddleware(adminController.ResetPassword)).Methods("POST")

	// Firestore collections API for admin
	router.HandleFunc("/admin/api/collections", adminController.AdminAuthMiddleware(docController.ListCollectionsHandler)).Methods("GET")
	router.HandleFunc("/admin/api/collections/{collection}/documents", adminController.AdminAuthMiddleware(docController.QueryDocumentsHandler)).Methods("GET")
	router.HandleFunc("/admin/api/collections/{collection}/documents", adminController.AdminAuthMiddleware(docController.CreateDocumentHandler)).Methods("POST")
	router.HandleFunc("/admin/api/collections/{collection}/documents/{id}", adminController.AdminAuthMiddleware(docController.GetDocumentHandler)).Methods("GET")
	router.HandleFunc("/admin/api/collections/{collection}/documents/{id}", adminController.AdminAuthMiddleware(docController.UpdateDocumentHandler)).Methods("PUT", "PATCH")
	router.HandleFunc("/admin/api/collections/{collection}/documents/{id}", adminController.AdminAuthMiddleware(docController.DeleteDocumentHandler)).Methods("DELETE")
}
