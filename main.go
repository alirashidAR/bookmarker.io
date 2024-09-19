// main.go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/joho/godotenv"
)

var db *pgxpool.Pool

type Bookmark struct {
    ID          int       `json:"id"`
    Title       string    `json:"title"`
    URL         string    `json:"url"`
    Description string    `json:"description"`
    Tags        string    `json:"tags"`
    CreatedAt   time.Time `json:"created_at"`
}

func initDB() {
    var err error

    err = godotenv.Load()
    if err != nil {
        log.Fatalf("Error loading .env file: %v\n", err)
    }

    connString := os.Getenv("DATABASE_URI")
    if connString == "" {
        log.Fatalf("DATABASE_URI environment variable is not set")
    }

    db, err = pgxpool.New(context.Background(), connString)
    if err != nil {
        log.Fatalf("Unable to connect to database: %v\n", err)
    }

    if err := db.Ping(context.Background()); err != nil {
        log.Fatalf("Unable to ping the database: %v\n", err)
    }

    fmt.Println("Connected to the database.")
}

func main() {
    initDB()
    defer db.Close()

    r := gin.Default()

    r.Static("/styles", "./styles")
    r.LoadHTMLGlob("templates/*")

    r.GET("/", getBookmarks)
    r.POST("/bookmarks", addBookmark)
    r.POST("/bookmarks/delete/:id", deleteBookmark)

    r.Run(":8080")
}

func getBookmarks(c *gin.Context) {
    rows, err := db.Query(context.Background(), "SELECT id, title, url, description, tags, created_at FROM bookmarks ORDER BY created_at DESC")
    if err != nil {
        c.String(http.StatusInternalServerError, "Error fetching bookmarks: %v", err)
        return
    }
    defer rows.Close()

    var bookmarks []Bookmark
    for rows.Next() {
        var bookmark Bookmark
        err := rows.Scan(&bookmark.ID, &bookmark.Title, &bookmark.URL, &bookmark.Description, &bookmark.Tags, &bookmark.CreatedAt)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error scanning row: %v", err)
            return
        }
        bookmarks = append(bookmarks, bookmark)
    }

    c.HTML(http.StatusOK, "index.html", gin.H{"bookmarks": bookmarks})
}

func addBookmark(c *gin.Context) {
    var bookmark Bookmark

    if err := c.Bind(&bookmark); err != nil {
        c.String(http.StatusBadRequest, "Invalid request data: %v", err)
        return
    }

    if bookmark.Title == "" || bookmark.URL == "" {
        c.String(http.StatusBadRequest, "Title and URL cannot be empty.")
        return
    }

    query := `INSERT INTO bookmarks (title, url, description, tags) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
    err := db.QueryRow(context.Background(), query, bookmark.Title, bookmark.URL, bookmark.Description, bookmark.Tags).Scan(&bookmark.ID, &bookmark.CreatedAt)
    if err != nil {
        c.String(http.StatusInternalServerError, "Error adding bookmark: %v", err)
        return
    }

    c.HTML(http.StatusOK, "bookmark.html", gin.H{"bookmark": bookmark})
}

func deleteBookmark(c *gin.Context) {
    id := c.Param("id")
    _, err := db.Exec(context.Background(), "DELETE FROM bookmarks WHERE id = $1", id)
    if err != nil {
        c.String(http.StatusInternalServerError, "Error deleting bookmark: %v", err)
        return
    }

    c.Status(http.StatusOK)
}
