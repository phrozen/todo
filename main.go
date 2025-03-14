package main

import (
	_ "embed"
	"flag"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//go:embed index.html
var index []byte

type Todo struct {
	ID        uint           `json:"id" gorm:"unique;primaryKey;autoIncrement"`
	Title     string         `json:"title"`
	Done      bool           `json:"done"`
	Due       time.Time      `json:"due"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type App struct {
	db  *gorm.DB
	srv *fiber.App
}

func NewApp(path string) (*App, error) {
	// Open a database connection
	// The database file will be created if it doesn't exist
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// Auto-migrate the Todo model
	// This will create the todos table in the database
	if err := db.AutoMigrate(&Todo{}); err != nil {
		return nil, err
	}
	// Set up the Fiber server
	srv := fiber.New()
	srv.Use(recover.New())
	srv.Use(requestid.New())
	srv.Use(logger.New())
	return &App{db: db, srv: srv}, nil
}

func (app *App) List(c *fiber.Ctx) error {
	var todos []Todo
	if err := app.db.Find(&todos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error retrieving todos"})
	}
	return c.JSON(todos)
}

func (app *App) Create(c *fiber.Ctx) error {
	todo := new(Todo)
	if err := c.BodyParser(todo); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	if err := app.db.Create(todo).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create todo"})
	}
	return c.JSON(todo)
}

func (app *App) Read(c *fiber.Ctx) error {
	var todo Todo
	if err := app.db.First(&todo, c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Todo not found"})
	}
	return c.JSON(todo)
}

func (app *App) Update(c *fiber.Ctx) error {
	var todo Todo
	if err := app.db.First(&todo, c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Todo not found"})
	}
	if err := c.BodyParser(&todo); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	if err := app.db.Save(&todo).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update todo"})
	}
	return c.JSON(todo)
}

func (app *App) Delete(c *fiber.Ctx) error {
	if err := app.db.Delete(&Todo{}, c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Todo not found"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func main() {
	// Parse command-line flags
	path := flag.String("db", "todo.db", "database file")
	addr := flag.String("addr", ":3000", "server address")
	devm := flag.Bool("dev", false, "development mode")
	flag.Parse()

	// Use the NewApp function to initialize the application
	app, err := NewApp(*path)
	if err != nil {
		log.Fatalf("failed to create new app: %v", err)
	}

	// Set up the root route to serve the embedded index.html
	app.srv.Get("/", func(c *fiber.Ctx) error {
		// Serve the embedded index.html file in development mode
		if *devm {
			return c.SendFile("index.html")
		}
		// Serve the embedded index.html file in production mode
		return c.Type("html").Send(index)
	})

	// Set up RESTful routes for Todo resources CRUD operations
	app.srv.Get("/todos", app.List)
	app.srv.Post("/todos", app.Create)
	app.srv.Get("/todos/:id", app.Read)
	app.srv.Put("/todos/:id", app.Update)
	app.srv.Delete("/todos/:id", app.Delete)

	// Start the Fiber server
	if err := app.srv.Listen(*addr); err != nil {
		log.Fatalf("server listen error: %v", err)
	}
}
