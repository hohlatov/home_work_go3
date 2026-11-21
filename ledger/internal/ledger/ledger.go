// Package ledger содержит бизнес-логику учёта транзакций
// и хранилище транзакций в памяти.
package ledger

import (
	"errors"
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
