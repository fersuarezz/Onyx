package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
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

	srcs, closers, err := multipartsToReaders(files)
	defer closeAll(closers)

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
	id, err := c.ParamsInt("id")
	if err != nil {
		return HandleError(fmt.Errorf("%w: id inválido", service.ErrInvalidInput))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	err = h.service.Delete(ctx, id)
	if err != nil {
		return HandleError(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Edit(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return HandleError(fmt.Errorf("%w: id inválido", service.ErrInvalidInput))
	}

	var req EditRequest

	err = c.BodyParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", service.ErrInvalidInput))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", service.ErrInvalidInput))
	}

	files := form.File["files"]
	srcs, closers, err := multipartsToReaders(files)
	defer closeAll(closers)

	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", service.ErrFileSystem))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	err = h.service.Edit(ctx, id, req.Date, req.Text, srcs)
	if err != nil {
		return HandleError(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) AllAttachments(c *fiber.Ctx) error {
	var req AllAttachmentsRequest

	err := c.QueryParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: query inválida", service.ErrInvalidInput))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	keys, hasMore, err := h.service.AllAttachments(ctx, req.Limit, req.Offset)
	if err != nil {
		return HandleError(err)
	}

	return c.Status(fiber.StatusOK).JSON(AllAttachmentsResponse{
		Keys:    keys,
		HasMore: hasMore,
	})
}

func (h *Handler) GetAttachment(c *fiber.Ctx) error {
	var req GetAttachmentRequest

	err := c.ParamsParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: params inválidos", service.ErrInvalidInput))
	}

	path, err := h.service.GetAttachmentPath(req.Key, req.Size)
	if err != nil {
		return HandleError(err)
	}

	c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	return c.SendFile(path)
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
	case errors.Is(err, service.ErrNotFound):
		slog.Error("Not found", "err", err)
		return fiber.NewError(fiber.StatusNotFound, "Archivo no encontrado")
	default:
		slog.Error("error no controlado", "err", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Error interno")
	}
}

func multipartsToReaders(multiparts []*multipart.FileHeader) ([]io.ReadSeeker, []io.Closer, error) {
	var seekers []io.ReadSeeker
	var closers []io.Closer

	for _, m := range multiparts {
		src, err := m.Open()
		if err != nil {
			return nil, closers, fmt.Errorf("%w: %w", service.ErrFileSystem, err)
		}
		closers = append(closers, src)
		seekers = append(seekers, src)
	}

	return seekers, closers, nil
}

func closeAll(closers []io.Closer) {
	for _, c := range closers {
		err := c.Close()
		if err != nil {
			slog.Error("no se pudo cerrar el archivo", "err", err)
		}
	}
}
