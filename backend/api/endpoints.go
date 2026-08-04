package api

import (
	"context"
	"errors"
	"log/slog"
	"onyx/core"
	"onyx/service"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *service.DiaryService
}

func NewHandler(s *service.DiaryService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Insert(c *fiber.Ctx) error {
	var req InsertRequest

	err := c.BodyParser(&req)

	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "cuerpo inválido")
	}

	entry := core.DiaryEntry{
		Date:        req.Date,
		Text:        req.Text,
		Attachments: req.Attachments,
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	id, err := h.service.Insert(ctx, entry)

	if err != nil {
		return HandleServiceError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(InsertResponse{
		CreatedId: id,
		Response:  "ok",
	})

}

func (h *Handler) GetByDate(c *fiber.Ctx) error {
	var req GetByDateRequest

	err := c.QueryParser(&req)

	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "parámetros inválidos")
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	entries, err := h.service.GetByDate(ctx, req.Date, req.IgnoreYear)

	if err != nil {
		return HandleServiceError(err)
	}

	return c.Status(fiber.StatusOK).JSON(GetByDateResponse{
		Entries:  entries,
		Response: "ok",
	})
}

func HandleServiceError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		return fiber.NewError(fiber.StatusBadRequest, "Datos de entrada inválidos")

	case errors.Is(err, context.DeadlineExceeded):
		slog.Error("timeout", "err", err)
		return fiber.NewError(fiber.StatusGatewayTimeout, "La petición tardó demasiado")

	case errors.Is(err, service.ErrDatabase):
		slog.Error("error de base de datos", "err", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Error interno")

	default:
		slog.Error("error no controlado", "err", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Error interno")
	}
}
