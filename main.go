package main

import (
	"awesomeProject/internal/handlers"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"os"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Get Telegram token
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("❌ TELEGRAM_TOKEN not set")
	}

	// Get database connection string
	dbConnStr := os.Getenv("DATABASE_URL")
	if dbConnStr == "" {
		dbConnStr = "host=localhost port=5432 user=postgres password=123 dbname=goalsbot sslmode=disable"
		log.Println("⚠️  Using default database connection string")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("❌ Database connection error: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err = db.Ping(); err != nil {
		log.Fatalf("❌ Cannot ping database: %v", err)
	}
	log.Println("✅ Connected to database")

	// Initialize repository and service
	repo := repository.NewRepository(db)
	svc := service.NewService(repo)

	// Initialize bot
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("❌ Bot initialization error: %v", err)
	}

	bot.Debug = false
	log.Printf("✅ Bot authorized as @%s", bot.Self.UserName)

	// Initialize handler
	handler := handlers.NewBotHandler(bot, svc)

	// Start receiving updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"}

	updates := bot.GetUpdatesChan(u)

	log.Println("🚀 Bot is running...")

	// Handle updates
	for update := range updates {
		handler.HandleUpdate(update)
	}
}
