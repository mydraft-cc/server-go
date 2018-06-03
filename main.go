package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/rs/xid"
	"github.com/spf13/viper"
)

var (
	bucket *storage.BucketHandle
	ctx    context.Context
)

// CreateResponse to return the write token
type CreateResponse struct {
	TokenWrite string `json:"writeToken"`
	TokenRead  string `json:"readToken"`
}

func setupConfig() error {
	viper.SetDefault("gce.storage.bucketName", "athene-diagrams")

	viper.SetDefault("web.port", ":4000")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigName("config")

	viper.AutomaticEnv()
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			log.Info("Cannot find config file: %s", err)
			return nil
		default:
			log.Fatalf("Fatal error config file: %s", err)
			return err
		}
	}

	return nil
}

func main() {
	ctx = context.Background()

	err := setupConfig()
	if err != nil {
		os.Exit(2)
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorf("Cannot connect to storage client", err)
		os.Exit(2)
	}

	bucket = client.Bucket(viper.GetString("gce.storage.bucketName"))

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes

	// Get a diagram by token
	e.GET("/:token", getDiagram)

	// Update a diagram with token and store token
	e.PUT("/:token/:writeToken", putDiagram)

	// Create a new diagram
	e.POST("/", postDiagram)

	// Run on port 4000 by default
	port := viper.GetString("web.port")

	// Start server
	e.Logger.Fatal(e.Start(port))
}

// Handlers

func getDiagram(c echo.Context) error {
	storageToken := c.Param("token")
	storageObj := bucket.Object(storageToken)

	reader, err := storageObj.NewReader(context.Background())
	// Return 404 if object does not exist.
	if err != nil {
		log.Error(err)
		return c.String(http.StatusNotFound, "Not found")
	}

	return c.Stream(http.StatusOK, "text/json", reader)
}

func putDiagram(c echo.Context) error {
	body := c.Request().Body

	if body == nil {
		return c.String(http.StatusBadRequest, "Bad Request")
	}
	defer body.Close()

	token := c.Param("token")

	attrs, err := bucket.Object(token).Attrs(ctx)
	// Return 404 if object does not exist.
	if err != nil {
		log.Error(err)
		return c.String(http.StatusNotFound, "Not found")
	}

	storeToken := c.Param("writeToken")
	// Check if the store token matchs the metadata of the object
	if attrs.Metadata["write-token"] != storeToken {
		return c.String(http.StatusForbidden, "Write token not valid")
	}

	wc := bucket.Object(token).NewWriter(ctx)

	if _, err := io.Copy(wc, body); err != nil {
		return err
	}

	if err := wc.Close(); err != nil {
		return err
	}

	return c.String(http.StatusOK, "Updated")
}

func postDiagram(c echo.Context) error {
	body := c.Request().Body

	if body == nil {
		return c.String(http.StatusBadRequest, "Bad Request")
	}
	defer body.Close()

	token := xid.New().String()
	tokenWrite := xid.New().String()

	wc := bucket.Object(token).NewWriter(ctx)
	wc.ContentType = "text/json"
	wc.Metadata = map[string]string{
		"write-token": tokenWrite,
	}

	if _, err := io.Copy(wc, body); err != nil {
		return err
	}

	if err := wc.Close(); err != nil {
		return err
	}

	response := &CreateResponse{
		TokenWrite: tokenWrite,
		TokenRead:  token,
	}

	return c.JSON(http.StatusCreated, response)
}
