package main

import (
	"fmt"
	"log"
	"time"

	"ledger/internal/ledger"
)

func main() {
	fmt.Println("Ledger service started")

	// Тестовые транзакции с разными категориями и суммами.
	testTransactions := []ledger.Transaction{
		{Amount: 1250.50, Category: "Продукты", Description: "Покупки в супермаркете", Date: date("2026-11-18")},
		{Amount: 350, Category: "Транспорт", Description: "Такси до университета", Date: date("2026-11-19")},
		{Amount: 4990, Category: "Образование", Description: "Подписка на курс по Go", Date: date("2026-11-20")},
		{Amount: 0, Category: "Прочее", Description: "Некорректная транзакция с нулевой суммой"},
	}

	for _, tx := range testTransactions {
		if err := ledger.AddTransaction(tx); err != nil {
			log.Printf("не удалось добавить транзакцию %q: %v", tx.Description, err)
		}
	}

	list := ledger.ListTransactions()
	fmt.Printf("\nСписок транзакций (%d):\n", len(list))
	fmt.Printf("%-4s %-12s %-12s %10s  %s\n", "ID", "Дата", "Категория", "Сумма", "Описание")
	for _, tx := range list {
		fmt.Printf("%-4d %-12s %-12s %10.2f  %s\n",
			tx.ID, tx.Date.Format("2006-01-02"), tx.Category, tx.Amount, tx.Description)
	}
}

// date разбирает дату в формате ГГГГ-ММ-ДД.
func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		log.Fatalf("некорректная дата %q: %v", s, err)
	}
	return t
}
