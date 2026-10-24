package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"onyx/domain"
	"onyx/service/diary"
	"time"

	"github.com/gofiber/fiber/v2"
)

type DiaryHandler struct {
	service *diary.Service
}

func NewDiaryHandler(svc *diary.Service) *DiaryHandler {
	return &DiaryHandler{service: svc}
}

func (h *DiaryHandler) Insert(c *fiber.Ctx) error {
	var req InsertRequest

	err := c.BodyParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", domain.ErrInvalidInput))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", domain.ErrInvalidInput))
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

func (h *DiaryHandler) GetByDate(c *fiber.Ctx) error {
	var req GetByDateRequest

	err := c.QueryParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: parámetros inválidos", domain.ErrInvalidInput))
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

func (h *DiaryHandler) Search(c *fiber.Ctx) error {
	var req SearchRequest

	err := c.QueryParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: parámetros inválidos", domain.ErrInvalidInput))
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

func (h *DiaryHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return HandleError(fmt.Errorf("%w: id inválido", domain.ErrInvalidInput))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	err = h.service.Delete(ctx, id)
	if err != nil {
		return HandleError(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *DiaryHandler) Edit(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return HandleError(fmt.Errorf("%w: id inválido", domain.ErrInvalidInput))
	}

	var req EditRequest

	err = c.BodyParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", domain.ErrInvalidInput))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", domain.ErrInvalidInput))
	}

	files := form.File["files"]
	srcs, closers, err := multipartsToReaders(files)
	defer closeAll(closers)

	if err != nil {
		return HandleError(fmt.Errorf("%w: cuerpo inválido", domain.ErrFileSystem))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	err = h.service.Edit(ctx, id, req.Date, req.Text, srcs)
	if err != nil {
		return HandleError(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *DiaryHandler) AllAttachments(c *fiber.Ctx) error {
	var req AllAttachmentsRequest

	err := c.QueryParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: query inválida", domain.ErrInvalidInput))
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

func (h *DiaryHandler) GetAttachment(c *fiber.Ctx) error {
	var req GetAttachmentRequest

	err := c.ParamsParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: params inválidos", domain.ErrInvalidInput))
	}

	file, err := h.service.GetAttachment(req.Key, req.Size)
	if err != nil {
		return HandleError(err)
	}

	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		return HandleError(fmt.Errorf("%w: %w", domain.ErrFileSystem, err))
	}

	c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	c.Set(fiber.HeaderContentType, "image/jpeg")
	return c.SendStream(file, int(size)) // Auto closes
}

func (h *DiaryHandler) GetAtemporal(c *fiber.Ctx) error {
	var req GetAtemporalEntriesRequest

	err := c.QueryParser(&req)
	if err != nil {
		return HandleError(fmt.Errorf("%w: parámetros inválidos", domain.ErrInvalidInput))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	entries, hasMore, err := h.service.GetAtemporal(ctx, req.Limit, req.Offset)
	if err != nil {
		return HandleError(err)
	}

	return c.Status(fiber.StatusOK).JSON(GetAtemporalEntriesResponse{
		Entries: entries,
		HasMore: hasMore,
	})
}

func (h *DiaryHandler) GetById(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return HandleError(fmt.Errorf("%w: parámetros inválidos", domain.ErrInvalidInput))
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*5)
	defer cancel()

	entry, err := h.service.GetById(ctx, id)

	if err != nil {
		return HandleError(err)
	}
	return c.Status(fiber.StatusOK).JSON(GetByIdResponse{
		Entry: entry,
	})
}

func HandleError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return fiber.NewError(fiber.StatusBadRequest, "Datos de entrada inválidos")
	case errors.Is(err, context.DeadlineExceeded):
		slog.Error("timeout", "err", err)
		return fiber.NewError(fiber.StatusGatewayTimeout, "La petición tardó demasiado")
	case errors.Is(err, domain.ErrDatabase):
		slog.Error("error de base de datos", "err", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Error interno")
	case errors.Is(err, domain.ErrFileSystem):
		slog.Error("error de file system", "err", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Error interno")
	case errors.Is(err, domain.ErrNotFound):
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
			return nil, closers, fmt.Errorf("%w: %w", domain.ErrFileSystem, err)
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

func (h *DiaryHandler) RegisterRoutes(r fiber.Router) {
	g := r.Group("/diary")
	g.Get("/search", h.Search)
	g.Get("/attachments", h.AllAttachments)
	g.Get("/attachments/:key/:size", h.GetAttachment)
	g.Get("/atemporal", h.GetAtemporal)
	g.Get("/:id", h.GetById)
	g.Post("/", h.Insert)
	g.Get("/", h.GetByDate)
	g.Delete("/:id", h.Delete)
	g.Put("/:id", h.Edit)
}
