// @title Firebase Firestore Forwarder API
// @version 1.0
// @description API for interacting with Firebase Firestore
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"firestore-admin-client/config"
	"firestore-admin-client/controller"
	_ "firestore-admin-client/docs"
	"firestore-admin-client/router"
	"log"

	firebase "firebase.google.com/go"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Warning: Error loading .env file: %v", err)
	}

	config := config.LoadConfig()

	ctx := context.Background()
	opt := option.WithCredentialsFile(config.FirebaseCredFile)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("Error initializing firebase app: %v", err)
	}

	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Error initializing firestore client: %v", err)
	}
	defer firestoreClient.Close()

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("Error initializing firebase auth client: %v", err)
	}

	authController := controller.NewAuthController(config)
	adminController := controller.NewAdminController(config, authClient)
	docController := controller.NewDocumentController(firestoreClient, authClient, config)

	r := router.SetupRouter(authController, adminController, docController)
	server := router.CreateServer(r, config)

	log.Printf("Server starting on port %s", config.Port)
	log.Fatal(server.ListenAndServe())
}
