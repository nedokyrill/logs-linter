# logs-linter

Статический анализатор для проверки качества лог-сообщений в Go-проектах. Интегрируется с golangci-lint и поддерживает стандартные библиотеки логирования.

## Возможности

Линтер проверяет лог-сообщения на соответствие следующим правилам:

1. **Строчная буква в начале** — сообщения должны начинаться с маленькой буквы
2. **Английский язык** — только английские сообщения разрешены
3. **Отсутствие спецсимволов** — запрещены эмодзи и множественные знаки препинания
4. **Безопасность данных** — обнаружение потенциально чувствительной информации

## Поддерживаемые библиотеки

- `log` — стандартная библиотека Go
- `log/slog` — структурированное логирование
- `go.uber.org/zap` — высокопроизводительный логгер

## Установка и использование

### Вариант 1: Как плагин для golangci-lint

1. Создайте файл `.custom-gcl.yml` в корне вашего проекта:

```yaml
version: v2.11.1
plugins:
  - module: 'logs-linter'
    import: 'logs-linter/pkg/plugin'
    version: latest
```

2. Соберите кастомный бинарник golangci-lint:

```bash
golangci-lint custom
```

3. Добавьте линтер в `.golangci.yml`:

```yaml
linters:
  enable:
    - logslinter
```

4. Запустите проверку:

```bash
./custom-gcl run ./...
```

### Вариант 2: Standalone использование

Соберите и запустите напрямую:

```bash
go build -o logs-linter ./cmd/logs-linter
./logs-linter ./...
```

Или установите через go install (если модуль опубликован):

```bash
go install github.com/nedokyrill/logs-linter/cmd/logs-linter@latest
```

Или из локальной директории:

```bash
go install ./cmd/logs-linter
```

## Примеры проверок

### Правило 1: Строчная буква

```go
// Плохо
log.Info("Starting application")
slog.Error("Connection failed")

// Хорошо
log.Info("starting application")
slog.Error("connection failed")
```

### Правило 2: Английский язык

```go
// Плохо
log.Info("приложение запущено")
slog.Debug("ошибка подключения")

// Хорошо
log.Info("application started")
slog.Debug("connection error")
```

### Правило 3: Спецсимволы

```go
// Плохо
log.Info("server ready! 🚀")
log.Warn("attention!!!")
slog.Error("failed...")

// Хорошо
log.Info("server ready")
log.Warn("attention required")
slog.Error("operation failed")
```

### Правило 4: Чувствительные данные

```go
// Плохо
password := "secret"
log.Info("password: " + password)
slog.Debug("api_key=" + key)

// Хорошо
log.Info("authentication successful")
slog.Debug("api request processed")
```

## Конфигурация

По умолчанию все правила включены. Вы можете настроить поведение через `.golangci.yml`:

```yaml
linters-settings:
  custom:
    logslinter:
      type: "module"
      description: "Linter for log messages"
```

### Сборка

```bash
make build
```

### Тестирование

```bash
make test
```

**Примечание:** Тесты могут требовать настройки golden файлов для `analysistest`. Это известная особенность работы с тестовыми данными анализаторов.

## Технические детали

Анализатор использует `golang.org/x/tools/go/analysis` для парсинга AST и поиска вызовов функций логирования. Проверка выполняется на этапе компиляции, что позволяет обнаруживать проблемы до запуска приложения.

## CI/CD

Проект использует GitHub Actions для автоматического тестирования и сборки. Статус сборки можно посмотреть в разделе Actions репозитория.