package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/joeshaw/envdecode"
	"github.com/teamzidi/gchat-devin/chat"
	"github.com/teamzidi/gchat-devin/devin"
	"github.com/teamzidi/gchat-devin/storage"
)

var env struct {
	Port    int    `env:"PORT,default=8080"`
	Project string `env:"PROJECT,required"`
}

func main() {
	if err := envdecode.Decode(&env); err != nil {
		log.Fatalf("envdecode error: %v", err)
	}

	if err := run(); err != nil {
		log.Fatalf("run error: %v", err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	devinAPIKey, err := getSecret(ctx, env.Project, "DEVIN_API_KEY")
	if err != nil {
		return fmt.Errorf("get devin api key: %w", err)
	}

	devinClient := devin.NewClient(devinAPIKey)

	store, err := storage.NewFirestoreStore(ctx, env.Project)
	if err != nil {
		return fmt.Errorf("firestore store: %w", err)
	}

	handler := chat.NewHandler(devinClient, store)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.HandleEvent)

	addr := fmt.Sprintf(":%d", env.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("listening on port %d", env.Port)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	return nil
}

func getSecret(ctx context.Context, project, name string) (string, error) {
	if value := os.Getenv(name); value != "" {
		return value, nil
	}

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("secret manager client: %w", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", project, name),
	}

	resp, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("access secret version: %w", err)
	}

	return string(resp.Payload.Data), nil
}
