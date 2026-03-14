package analyzer

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "logslinter",
	Doc:  "check log messages for common issues",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Проверяем вызовы функций логирования
			checkLogCall(pass, callExpr)
			return true
		})
	}
	return nil, nil
}

func checkLogCall(pass *analysis.Pass, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Получаем имя функции (Info, Error, Debug, Warn и т.д.)
	fnName := sel.Sel.Name
	if !isLogFunction(fnName) {
		return
	}

	// Проверяем аргументы
	if len(call.Args) == 0 {
		return
	}

	var msgArg ast.Expr
	var msgPos token.Pos

	// Проверяем тип селектора
	switch x := sel.X.(type) {
	case *ast.Ident:
		// Прямой вызов: slog.Info(), log.Print(), zap.Info()
		pkgName := x.Name
		if !isLogPackage(pkgName) {
			return
		}

		msgArg, msgPos = findMessageArg(call, pkgName, fnName)
	case *ast.SelectorExpr:
		// Вызов через селектор: logger.Info() где logger может быть экземпляром zap.Logger
		// Проверяем, что это вызов метода логгера
		if isLogFunction(fnName) {
			// Для zap: logger.Info("message")
			msgArg, msgPos = findMessageArg(call, "", fnName)
		}
	default:
		// Другие типы вызовов пока не поддерживаем
		return
	}

	if msgArg == nil {
		return
	}

	// Извлекаем строку из литерала
	msg := extractString(msgArg)
	if msg == "" {
		return
	}

	// Проверяем правила
	checkRules(pass, msg, msgPos)
}

func findMessageArg(call *ast.CallExpr, pkgName, fnName string) (ast.Expr, token.Pos) {
	if len(call.Args) == 0 {
		return nil, 0
	}

	// Для slog: slog.Info("message") или slog.Info(ctx, "message")
	if pkgName == "slog" {
		// Проверяем первый аргумент
		if isStringLiteral(call.Args[0]) {
			return call.Args[0], call.Args[0].Pos()
		}
		// Если первый аргумент не строка, проверяем второй (может быть контекст)
		if len(call.Args) > 1 && isStringLiteral(call.Args[1]) {
			return call.Args[1], call.Args[1].Pos()
		}
	}

	// Для zap: zap.Info("message") или logger.Info("message")
	if pkgName == "zap" || strings.HasSuffix(fnName, "Info") || strings.HasSuffix(fnName, "Error") ||
		strings.HasSuffix(fnName, "Debug") || strings.HasSuffix(fnName, "Warn") ||
		strings.HasSuffix(fnName, "Fatal") || strings.HasSuffix(fnName, "Panic") {
		if isStringLiteral(call.Args[0]) {
			return call.Args[0], call.Args[0].Pos()
		}
	}

	// Для log: log.Print("message"), log.Printf("format", args)
	if pkgName == "log" {
		if fnName == "Printf" || fnName == "Println" || fnName == "Print" ||
			fnName == "Fatal" || fnName == "Fatalf" || fnName == "Fatalln" ||
			fnName == "Panic" || fnName == "Panicf" || fnName == "Panicln" {
			if isStringLiteral(call.Args[0]) {
				return call.Args[0], call.Args[0].Pos()
			}
		}
	}

	return nil, 0
}

func isLogFunction(name string) bool {
	logFunctions := []string{
		"Info", "Error", "Debug", "Warn", "Warning", "Fatal", "Panic",
		"Print", "Printf", "Println",
		"InfoContext", "ErrorContext", "DebugContext", "WarnContext",
	}
	for _, fn := range logFunctions {
		if name == fn {
			return true
		}
	}
	return false
}

func isLogPackage(name string) bool {
	return name == "log" || name == "slog" || name == "zap"
}

func isStringLiteral(expr ast.Expr) bool {
	_, ok := expr.(*ast.BasicLit)
	if ok {
		return true
	}
	// Проверяем бинарные операции конкатенации строк
	binary, ok := expr.(*ast.BinaryExpr)
	if ok && binary.Op == token.ADD {
		return isStringLiteral(binary.X) || isStringLiteral(binary.Y)
	}
	return false
}

func extractString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			// Убираем кавычки
			unquoted := strings.Trim(e.Value, `"`)
			unquoted = strings.Trim(unquoted, "`")
			// Убираем экранированные кавычки
			unquoted = strings.ReplaceAll(unquoted, `\"`, `"`)
			return unquoted
		}
	case *ast.BinaryExpr:
		if e.Op == token.ADD {
			left := extractString(e.X)
			right := extractString(e.Y)
			// Если одна из частей пустая, возвращаем другую
			if left == "" {
				return right
			}
			if right == "" {
				return left
			}
			return left + right
		}
	}
	return ""
}

func checkRules(pass *analysis.Pass, msg string, pos token.Pos) {
	// Правило 1: Сообщение должно начинаться со строчной буквы
	if !checkLowercaseStart(msg) {
		pass.Reportf(pos, "log message should start with lowercase letter: %q", msg)
	}

	// Правило 2: Сообщение должно быть на английском языке
	if !checkEnglishOnly(msg) {
		pass.Reportf(pos, "log message should be in English only: %q", msg)
	}

	// Правило 3: Сообщение не должно содержать спецсимволы или эмодзи
	if !checkNoSpecialChars(msg) {
		pass.Reportf(pos, "log message should not contain special characters or emojis: %q", msg)
	}

	// Правило 4: Сообщение не должно содержать чувствительные данные
	if !checkNoSensitiveData(msg) {
		pass.Reportf(pos, "log message should not contain potentially sensitive data: %q", msg)
	}
}

// Правило 1: Проверка на строчную букву в начале
func checkLowercaseStart(msg string) bool {
	if len(msg) == 0 {
		return true
	}
	firstRune := []rune(msg)[0]
	return unicode.IsLower(firstRune) || !unicode.IsLetter(firstRune)
}

// Правило 2: Проверка на английский язык
func checkEnglishOnly(msg string) bool {
	for _, r := range msg {
		// Пропускаем пробелы, цифры, знаки препинания и ASCII символы
		if unicode.IsSpace(r) || unicode.IsDigit(r) || unicode.IsPunct(r) || r < 128 {
			continue
		}
		// Проверяем, является ли символ кириллицей или другими не-латинскими буквами
		if unicode.Is(unicode.Cyrillic, r) || (unicode.IsLetter(r) && !isLatin(r)) {
			return false
		}
	}
	return true
}

func isLatin(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// Правило 3: Проверка на спецсимволы и эмодзи
func checkNoSpecialChars(msg string) bool {
	// Проверяем на множественные восклицательные или вопросительные знаки
	if strings.Contains(msg, "!!!") || strings.Contains(msg, "???") ||
		strings.Contains(msg, "!!") || strings.Contains(msg, "??") {
		return false
	}

	// Разрешенные символы: буквы, цифры, пробелы, основные знаки препинания
	allowedPunct := map[rune]bool{
		'.': true, ',': true, ':': true, ';': true,
		'-': true, '_': true,
		'(': true, ')': true, '[': true, ']': true,
		'{': true, '}': true, '=': true,
		'!': true, '?': true, // Разрешены по одному
	}

	runes := []rune(msg)
	skipNext := 0
	for i, r := range runes {
		if skipNext > 0 {
			skipNext--
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			continue
		}
		if unicode.IsPunct(r) {
			// Многоточие разрешено
			if r == '.' && i+2 < len(runes) && runes[i+1] == '.' && runes[i+2] == '.' {
				// Пропускаем все три точки многоточия
				skipNext = 2
				continue
			}
			if allowed, ok := allowedPunct[r]; ok {
				if !allowed {
					return false
				}
			} else {
				// Неизвестный знак препинания - считаем спецсимволом
				return false
			}
		} else if !unicode.IsPrint(r) {
			// Непечатные символы
			return false
		} else if r > 127 {
			// Проверяем, является ли это эмодзи
			if isEmoji(r) {
				return false
			}
		}
	}
	return true
}

func isEmoji(r rune) bool {
	// Проверяем диапазоны эмодзи в Unicode
	return (r >= 0x1F300 && r <= 0x1F9FF) || // Miscellaneous Symbols and Pictographs
		(r >= 0x2600 && r <= 0x26FF) ||       // Miscellaneous Symbols
		(r >= 0x2700 && r <= 0x27BF) ||       // Dingbats
		(r >= 0xFE00 && r <= 0xFE0F) ||       // Variation Selectors
		(r >= 0x1F900 && r <= 0x1F9FF) ||     // Supplemental Symbols and Pictographs
		(r >= 0x1F1E0 && r <= 0x1F1FF)        // Regional Indicator Symbols
}

// Правило 4: Проверка на чувствительные данные
func checkNoSensitiveData(msg string) bool {
	msgLower := strings.ToLower(msg)
	sensitiveKeywords := []string{
		"password", "passwd", "pwd",
		"api_key", "apikey",
		"access_token", "refresh_token",
		"secret_key",
		"private_key",
		"ssh_key",
	}

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(msgLower, keyword) {
			// Проверяем паттерны, которые указывают на вывод чувствительных данных
			// password: value, password=value
			if strings.Contains(msgLower, keyword+":") ||
				strings.Contains(msgLower, keyword+"=") {
				return false
			}
		}
	}

	// Отдельная проверка для "token" - только если есть паттерн "token: " или "token="
	if strings.Contains(msgLower, "token") {
		if strings.Contains(msgLower, "token:") || strings.Contains(msgLower, "token=") {
			return false
		}
	}

	return true
}
