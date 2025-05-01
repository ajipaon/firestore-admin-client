package router

import (
	"firestore-admin-client/controller"

	"github.com/gorilla/mux"
)

func SetupAuthRoutes(router *mux.Router, authController *controller.AuthController) {
	// Login page
	router.HandleFunc("/login", authController.ServeLoginPage).Methods("GET")

	// Authentication endpoints
	router.HandleFunc("/auth/login", authController.HandleLogin).Methods("POST")
	router.HandleFunc("/auth/login/json", authController.HandleJSONLogin).Methods("POST")

	// Registration endpoint
	router.HandleFunc("/auth/register/json", authController.HandleJSONRegister).Methods("POST")
}
