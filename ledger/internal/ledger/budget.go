package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Budget — лимит расходов на категорию.
type Budget struct {
	Category string  `json:"category"` // категория, на которую выделен бюджет
	Limit    float64 `json:"limit"`    // лимит бюджета
}

// Ошибки, связанные с бюджетами.
var (
	ErrBudgetExceeded = errors.New("budget exceeded")
	ErrInvalidLimit   = errors.New("лимит бюджета должен быть больше 0")
)

// Хранилище бюджетов: ключ — нормализованное название категории.
// Защищено тем же мьютексом mu, что и транзакции, чтобы проверка бюджета
// и добавление транзакции выполнялись атомарно.
var budgets = make(map[string]Budget)

// categoryKey приводит название категории к единому виду,
// чтобы «Еда», «еда» и « еда » считались одной категорией.
func categoryKey(category string) string {
	return strings.ToLower(strings.TrimSpace(category))
}

// validateBudget проверяет корректность бюджета.
func validateBudget(b Budget) error {
	if categoryKey(b.Category) == "" {
		return ErrEmptyCategory
	}
	if b.Limit <= 0 {
		return fmt.Errorf("%w (категория %q, лимит %.2f)", ErrInvalidLimit, b.Category, b.Limit)
	}
	return nil
}

// SetBudget добавляет новый бюджет или обновляет лимит существующего.
func SetBudget(b Budget) error {
	if err := validateBudget(b); err != nil {
		return err
	}
	b.Category = strings.TrimSpace(b.Category)

	mu.Lock()
	defer mu.Unlock()
	budgets[categoryKey(b.Category)] = b
	return nil
}

// GetBudget возвращает бюджет категории и признак его наличия.
func GetBudget(category string) (Budget, bool) {
	mu.RLock()
	defer mu.RUnlock()
	b, ok := budgets[categoryKey(category)]
	return b, ok
}

// ListBudgets возвращает все бюджеты, отсортированные по категории.
func ListBudgets() []Budget {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]Budget, 0, len(budgets))
	for _, b := range budgets {
		result = append(result, b)
	}
	sort.Slice(result, func(i, j int) bool {
		return categoryKey(result[i].Category) < categoryKey(result[j].Category)
	})
	return result
}

// LoadBudgets читает из r JSON-массив бюджетов и добавляет их через SetBudget.
//
// Формат:
//
//	[{"category": "Еда", "limit": 5000}, {"category": "Транспорт", "limit": 3000}]
//
// Сначала проверяются все элементы. Если хотя бы один некорректен, бюджеты
// не применяются, чтобы хранилище не осталось заполненным наполовину.
func LoadBudgets(r io.Reader) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields() // опечатка вроде "limt" должна давать ошибку, а не нулевой лимит

	var list []Budget
	if err := dec.Decode(&list); err != nil {
		return describeJSONError(err)
	}
	// После массива не должно быть других данных.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("загрузка бюджетов: после JSON-массива есть лишние данные")
	}
	if list == nil {
		return errors.New("загрузка бюджетов: ожидался JSON-массив, получено null")
	}

	for i, b := range list {
		if err := validateBudget(b); err != nil {
			return fmt.Errorf("загрузка бюджетов: элемент #%d: %w", i+1, err)
		}
	}
	for i, b := range list {
		if err := SetBudget(b); err != nil {
			return fmt.Errorf("загрузка бюджетов: элемент #%d: %w", i+1, err)
		}
	}
	return nil
}

// describeJSONError переводит ошибки encoding/json в понятные сообщения.
func describeJSONError(err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError

	switch {
	case errors.Is(err, io.EOF):
		return errors.New("загрузка бюджетов: пустые входные данные")
	case errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("загрузка бюджетов: JSON обрывается раньше конца")
	case errors.As(err, &syntaxErr):
		return fmt.Errorf("загрузка бюджетов: синтаксическая ошибка JSON на позиции %d: %w", syntaxErr.Offset, err)
	case errors.As(err, &typeErr):
		if typeErr.Field == "" {
			return fmt.Errorf("загрузка бюджетов: ожидался JSON-массив объектов, получено %s", typeErr.Value)
		}
		// В новых версиях Go путь содержит индекс элемента массива: "0.limit".
		where, field := "", typeErr.Field
		if idx, rest, ok := strings.Cut(field, "."); ok {
			if n, err := strconv.Atoi(idx); err == nil {
				where, field = fmt.Sprintf("элемент #%d: ", n+1), rest
			}
		}
		return fmt.Errorf("загрузка бюджетов: %sполе %q должно иметь тип %s, получено %s",
			where, field, typeErr.Type, typeErr.Value)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return fmt.Errorf("загрузка бюджетов: неизвестное поле %s (допустимы: category, limit)", field)
	default:
		return fmt.Errorf("загрузка бюджетов: %w", err)
	}
}
