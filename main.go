package main

import (
	_ "embed"
	"flag"
	"log"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//go:embed index.html
var index []byte

type Todo struct {
	gorm.Model
	Title string `json:"title"`
	Done  bool   `json:"done"`
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
	// Create a new App instance
	app := &App{db: db}
	// Set up the Fiber server
	srv := fiber.New()
	// Set up the root route to serve the embedded index.html
	srv.Get("/", func(c *fiber.Ctx) error {
		return c.Type("html").Send(index)
	})
	// Set up RESTful routes for Todo resources CRUD operations
	srv.Get("/todos", app.List)
	srv.Post("/todos", app.Create)
	srv.Get("/todos/:id", app.Read)
	srv.Put("/todos/:id", app.Update)
	srv.Delete("/todos/:id", app.Delete)
	// Return the App instance
	app.srv = srv
	return app, nil
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
	flag.Parse()

	// Use the NewApp function to initialize the application
	app, err := NewApp(*path)
	if err != nil {
		log.Fatalf("failed to create new app: %v", err)
	}

	// Start the Fiber server
	if err := app.srv.Listen(*addr); err != nil {
		log.Fatalf("server listen error: %v", err)
	}
}
