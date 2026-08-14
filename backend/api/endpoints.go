package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"onyx/core/db"
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
		return HandleError(fmt.Errorf("%w: cuerpo inválido", service.ErrInvalidInput))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", service.ErrInvalidInput))
	}

	files := form.File["files"]

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	srcs, err := multipartsToReaders(files)
	if err != nil {
		return HandleError(err)
	}

	id, err := h.service.Insert(ctx, req.Date, req.Text, srcs)

	if err != nil {
		return HandleError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(InsertResponse{
		CreatedId: id,
	})
}

func (h *Handler) GetByDate(c *fiber.Ctx) error {
	var req GetByDateRequest

	err := c.QueryParser(&req)

	if err != nil {
		return HandleError(fmt.Errorf("%w: parámetros inválidos", service.ErrInvalidInput))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	entries, err := h.service.GetByDate(ctx, req.Date, req.IgnoreYear)

	if err != nil {
		return HandleError(err)
	}

	return c.Status(fiber.StatusOK).JSON(GetByDateResponse{
		Entries: entries,
	})
}

func (h *Handler) Search(c *fiber.Ctx) error {
	var req SearchRequest

	err := c.QueryParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: parámetros inválidos", service.ErrInvalidInput))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	entries, hasMore, err := h.service.Search(ctx, req.Text, req.Limit, req.Offset)

	if err != nil {
		return HandleError(err)
	}

	return c.Status(fiber.StatusOK).JSON(SearchResponse{
		Entries: entries,
		HasMore: hasMore,
	})
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	var req DeleteRequest

	err := c.BodyParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: parámetros inválidos", service.ErrInvalidInput))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	err = h.service.Delete(ctx, req.Id)
	if err != nil {
		return HandleError(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Edit(c *fiber.Ctx) error {
	var req EditRequest

	err := c.BodyParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", service.ErrInvalidInput))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", service.ErrInvalidInput))
	}

	files := form.File["files"]
	srcs, err := multipartsToReaders(files)
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", service.ErrInvalidInput))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	err = h.service.Edit(ctx, req.Id, req.Date, req.Text, srcs)
	if err != nil {
		return HandleError(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func HandleError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		return fiber.NewError(fiber.StatusBadRequest, "Datos de entrada inválidos")
	case errors.Is(err, context.DeadlineExceeded):
		slog.Error("timeout", "err", err)
		return fiber.NewError(fiber.StatusGatewayTimeout, "La petición tardó demasiado")
	case errors.Is(err, service.ErrDatabase):
		slog.Error("error de base de datos", "err", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Error interno")
	case errors.Is(err, service.ErrFileSystem):
		slog.Error("error de file system", "err", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Error interno")
	case errors.Is(err, db.ErrNotFound):
		slog.Error("DB 404", "err", err)
		return fiber.NewError(fiber.StatusNotFound, "Archivo no encontrado")
	default:
		slog.Error("error no controlado", "err", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Error interno")
	}
}

func multipartsToReaders(multiparts []*multipart.FileHeader) ([]io.ReadSeeker, error) {
	var seekers []io.ReadSeeker

	for _, m := range multiparts {
		src, err := m.Open()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", service.ErrFileSystem, err)
		}
		defer src.Close()
		seekers = append(seekers, src)
	}

	return seekers, nil
}
