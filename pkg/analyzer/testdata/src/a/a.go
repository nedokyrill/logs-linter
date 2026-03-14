package a

import (
	"log"
	"log/slog"
)

func testLog() {
	// Правило 1: Начинается с заглавной буквы - ошибка
	log.Print("Starting server")
	slog.Info("Starting server")

	// Правило 1: Правильно - начинается со строчной
	log.Print("starting server")
	slog.Info("starting server")

	// Правило 2: Русский язык - ошибка
	log.Print("запуск сервера")
	slog.Error("ошибка подключения")

	// Правило 2: Правильно - английский
	log.Print("starting server")
	slog.Error("failed to connect")

	// Правило 3: Спецсимволы и эмодзи - ошибка
	log.Print("server started!🚀")
	slog.Warn("warning: something went wrong...")
	log.Print("connection failed!!!")

	// Правило 3: Правильно - без спецсимволов
	log.Print("server started")
	slog.Warn("something went wrong")
	log.Print("connection failed")

	// Правило 4: Чувствительные данные - ошибка
	password := "secret123"
	log.Print("user password: " + password)
	slog.Debug("api_key=" + "key123")
	log.Print("token: " + "token123")

	// Правило 4: Правильно - без чувствительных данных
	log.Print("user authenticated successfully")
	slog.Debug("api request completed")
	log.Print("token validated")
}
