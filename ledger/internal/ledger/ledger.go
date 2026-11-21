// Package ledger содержит бизнес-логику учёта транзакций
// и хранилище транзакций в памяти.
package ledger

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// Transaction — финансовая транзакция.
type Transaction struct {
	ID          int       // уникальный идентификатор (автоинкремент)
	Amount      float64   // сумма
	Category    string    // категория траты
	Description string    // описание
	Date        time.Time // дата транзакции
}

// Ошибки валидации.
var (
	ErrZeroAmount    = errors.New("сумма транзакции не может быть равна 0")
	ErrEmptyCategory = errors.New("категория транзакции не может быть пустой")
)

// Хранилище транзакций в памяти. Инициализируется пустым срезом при запуске.
var (
	mu           sync.RWMutex
	transactions = make([]Transaction, 0)
)

// AddTransaction проверяет транзакцию и добавляет её в хранилище.
// ID назначается автоматически (len(transactions)+1).
// Если дата не указана, подставляется текущее время.
//
// Если для категории задан бюджет, сумма уже сохранённых транзакций этой
// категории вместе с новой не должна превышать лимит. Иначе возвращается
// ошибка, которая оборачивает ErrBudgetExceeded, и транзакция не сохраняется.
func AddTransaction(tx Transaction) error {
	if tx.Amount == 0 {
		return ErrZeroAmount
	}
	if strings.TrimSpace(tx.Category) == "" {
		return ErrEmptyCategory
	}
	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}

	mu.Lock()
	defer mu.Unlock()

	// Проверка бюджета и добавление выполняются под одной блокировкой,
	// чтобы две параллельные транзакции не превысили лимит вместе.
	if b, ok := budgets[categoryKey(tx.Category)]; ok {
		spent := spentLocked(tx.Category)
		if exceedsLimit(spent+tx.Amount, b.Limit) {
			return fmt.Errorf("%w: категория %q, лимит %.2f, уже потрачено %.2f, транзакция %.2f (доступно %.2f)",
				ErrBudgetExceeded, b.Category, b.Limit, spent, tx.Amount, b.Limit-spent)
		}
	}

	tx.ID = len(transactions) + 1
	transactions = append(transactions, tx)
	return nil
}

// ListTransactions возвращает копию всех сохранённых транзакций,
// чтобы вызывающий код не мог изменить внутреннее хранилище.
func ListTransactions() []Transaction {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]Transaction, len(transactions))
	copy(result, transactions)
	return result
}

// SpentByCategory возвращает сумму всех транзакций категории.
func SpentByCategory(category string) float64 {
	mu.RLock()
	defer mu.RUnlock()
	return spentLocked(category)
}

// exceedsLimit сравнивает суммы с точностью до копеек. Так погрешность float64
// не помешает уложиться ровно в лимит (например, 0.1 + 0.2 при лимите 0.3).
func exceedsLimit(total, limit float64) bool {
	return math.Round(total*100) > math.Round(limit*100)
}

// spentLocked считает сумму транзакций категории. Вызывающий код должен держать mu.
func spentLocked(category string) float64 {
	key := categoryKey(category)
	var total float64
	for _, tx := range transactions {
		if categoryKey(tx.Category) == key {
			total += tx.Amount
		}
	}
	return total
}
