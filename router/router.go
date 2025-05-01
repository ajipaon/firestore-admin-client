package router

import (
	"firestore-admin-client/config"
	"firestore-admin-client/controller"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SetupRouter(authController *controller.AuthController, adminController *controller.AdminController, docController *controller.DocumentController) *mux.Router {

	router := mux.NewRouter()

	fs := http.FileServer(http.Dir("static"))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	}).Methods("GET")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		controller.RespondWithJSON(w, http.StatusOK, controller.GenericResponse{
			Success: true,
			Message: "Server is healthy",
		})
	}).Methods("GET")

	SetupAuthRoutes(router, authController)
	SetupAdminRoutes(router, adminController, docController)
	SetupAPIRoutes(router, docController)

	return router
}

func CreateServer(router *mux.Router, config config.Config) *http.Server {
	return &http.Server{
		Handler:      router,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		Addr:         ":" + config.Port,
	}
}
