package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dhruv15803/social-media-app/cloudinary"
	"github.com/dhruv15803/social-media-app/db"
	"github.com/dhruv15803/social-media-app/handlers"
	"github.com/dhruv15803/social-media-app/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	DbConnStr string
	ClientUrl string
}

func loadConfig() (*Config, error) {

	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	port := ":" + os.Getenv("PORT")
	dbConnStr := os.Getenv("DB_CONN")
	clientUrl := os.Getenv("CLIENT_URL")

	return &Config{
		Port:      port,
		DbConnStr: dbConnStr,
		ClientUrl: clientUrl,
	}, nil
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load server config :- %v\n", err.Error())
	}

	db, err := db.ConnectToPostgresDb(config.DbConnStr)
	if err != nil {
		log.Fatalf("failed to establish connection to postgres db :- %v\n", err.Error())
	}

	defer db.Close()

	log.Println("successfully connected to Db")

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{config.ClientUrl},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// loading cloudinary instant
	cld, err := cloudinary.LoadCloudinaryInstance()

	if err != nil {
		log.Fatalf("failed to load cloudinary instance :- %v\n", err.Error())
	}

	storage := storage.NewStorage(db)             // storage layer
	handler := handlers.NewHandler(*storage, cld) // handler layer using the storage layer

	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Get("/health", handler.HealthCheckHandler)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", handler.RegisterUserHandler)
			r.Post("/login", handler.LoginUserHandler)
			r.With(handler.AuthMiddleware).Get("/user", handler.GetAuthUserHandler)
		})

		r.Route("/post", func(r chi.Router) {
			r.Get("/posts", handler.GetPublicPostsHandler)
			r.Get("/{postId}/comments", handler.GetPostCommentsHandler)
			r.Get("/{postId}/likes", handler.GetPostLikesHandler)
			r.Get("/{postId}/bookmarks", handler.GetPostBookmarksHandler)
			r.Get("/{postId}", handler.GetPostHandler)
			r.Get("/{postId}/metadata", handler.GetPostWithMetaDataHandler)

			r.Group(func(r chi.Router) {
				r.Use(handler.AuthMiddleware)
				r.Get("/feed", handler.GetPostsHandler)
				r.Get("/my-posts", handler.GetMyPostsHandler)
				r.Get("/my-liked-posts", handler.GetMyLikedPostsHandler)
				r.Post("/", handler.CreatePostHandler)
				r.Post("/{parentPostId}", handler.CreateChildPostHandler)
				r.Delete("/{postId}", handler.DeletePostHandler)
				r.Post("/{postId}/like", handler.LikePostHandler)
				r.Post("/{postId}/bookmark", handler.BookmarkPostHandler)
			})
		})

		r.Route("/user", func(r chi.Router) {
			r.Get("/{userId}/profile", handler.GetUserProfileHandler)
			r.With(handler.OptionalAuthMiddleware).Get("/{userId}/posts", handler.GetUserPostsHandler)
			r.With(handler.OptionalAuthMiddleware).Get("/{userId}/liked-posts", handler.GetUserLikedPostsHandler)
			r.With(handler.OptionalAuthMiddleware).Get("/{userId}/bookmarked-posts", handler.GetUserBookmarkedPostsHandler)
			r.With(handler.OptionalAuthMiddleware).Get("/{userId}/followers", handler.GetUserFollowersHandler)
			r.With(handler.OptionalAuthMiddleware).Get("/{userId}/followings", handler.GetUserFollowingsHandler)
			r.Get("/search", handler.GetSearchResultsHandler)

			r.Group(func(r chi.Router) {
				r.Use(handler.AuthMiddleware)
				r.Get("/notifications", handler.GetNotificationsHandler)
				r.Put("/", handler.UpdateUserHandler)
				r.Post("/{userId}/follow-request", handler.FollowRequestHandler)
				r.Post("/{userId}/follow", handler.FollowUserHandler)
				r.Post("/{userId}/follow-request/accept", handler.AcceptFollowRequestHandler)
				r.Get("/my-requests-sent", handler.GetFollowRequestsSentHandler)
				r.Get("/my-requests-received", handler.GetRequestsReceivedHandler)
				r.Get("/my-followings", handler.GetFollowingsHandler)
			})
		})

		r.Route("/file", func(r chi.Router) {
			r.Post("/upload", handler.UploadFileHandler)
		})
	})

	server := http.Server{
		Addr:         config.Port,
		Handler:      r,
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
		IdleTimeout:  time.Second * 30,
	}

	log.Printf("server listening on port %s", config.Port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start server on port %s , %v\n", config.Port, err.Error())
	}
}
