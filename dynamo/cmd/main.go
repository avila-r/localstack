package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"time"

	"github.com/avila-r/localstack/dynamo/posts"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gofiber/fiber/v2"
)

var (
	ctx = context.TODO()
)

func main() {
	_ = posts.Post{
		Id:                  "1",
		Title:               "my post",
		Content:             "post content",
		Status:              "posted",
		CreateTimestamp:     time.Now().Format("2006-01-02T15:04:05.000Z"),
		LastUpdateTimestamp: time.Now().Format("2006-01-02T15:04:05.000Z"),
	}

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: transport,
	}

	cfg, err = config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
		config.WithHTTPClient(client),
	)

	dynamo := dynamodb.NewFromConfig(cfg)

	service := posts.Service(dynamo, "posts_table")

	handler := posts.PostHandler{
		Service: service,
	}

	app := fiber.New()

	app.Get("/health", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{
			"message": "OK",
		})
	})

	app.Get("/post/:id", handler.Find)

	app.Post("/post", handler.Insert)

	if err := app.Listen(":80"); err != nil {
		panic(err)
	}
}
