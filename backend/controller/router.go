package controller

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type Routable interface {
	RegisterRoutes(r fiber.Router)
}

func CreateApp(modules ...Routable) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Onyx",
		BodyLimit:    64 * 1024 * 1024, // 64MB
		ErrorHandler: errorHandler,
	})

	// Middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	app.Use(requestid.New()) // Identificador único por petición
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${locals:requestid} | ${error}\n",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	v1 := app.Group("/api/v1")
	for _, m := range modules {
		m.RegisterRoutes(v1)
	}

	return app
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "Error interno"

	var fe *fiber.Error
	if errors.As(err, &fe) {
		code, msg = fe.Code, fe.Message
	}

	return c.Status(code).JSON(fiber.Map{"error": msg})
}
