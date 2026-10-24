package main

import (
	"onyx/controller"
	"onyx/repository/postgres"
	"onyx/repository/storage"
	"onyx/service/diary"

	"github.com/gofiber/fiber/v2"
)

func buildApp(pool postgres.Querier, mediaRoot string) *fiber.App {
	fileStorage := composeStorage(mediaRoot)
	diaryRepo := composeDiaryRepository(pool)

	diaryService := diary.NewService(diaryRepo, fileStorage)
	diaryHandler := controller.NewDiaryHandler(diaryService)

	return controller.CreateApp(diaryHandler)
}

// COMPOSERS: Añadir variable en .env en un futuro para elegir
func composeStorage(mediaRoot string) diary.Storage {
	return storage.NewFileSystem(mediaRoot)
}

func composeDiaryRepository(pool postgres.Querier) diary.Repository {
	return postgres.NewDiaryRepository(pool)
}
