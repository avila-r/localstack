package posts

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
)

type PostHandler struct {
	Service PostService
}

func (h *PostHandler) Find(c *fiber.Ctx) error {
	id := c.Params("id")

	post, err := h.Service.Find(id)
	if handle(c, err, fiber.StatusBadRequest) {
		return nil
	}

	return c.Status(fiber.StatusOK).JSON(post)
}

func (h *PostHandler) Insert(c *fiber.Ctx) error {
	var post Post

	if err := json.Unmarshal(c.Body(), &post); handle(c, err, fiber.StatusInternalServerError) {
		return nil
	}

	if post.CreateTimestamp == "" {
		post.CreateTimestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
	}

	if post.LastUpdateTimestamp == "" {
		post.LastUpdateTimestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
	}

	if _, err := h.Service.Insert(post); handle(c, err, fiber.StatusInternalServerError) {
		return nil
	}

	return c.Status(fiber.StatusOK).JSON(post)
}

func handle(c *fiber.Ctx, err error, status int) bool {
	if err != nil {
		c.Status(status).JSON(fiber.Map{"error": err.Error()})
		return true
	}
	return false
}
