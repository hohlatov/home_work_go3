package ledger

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func mustSetBudget(t *testing.T, b Budget) {
	t.Helper()
	if err := SetBudget(b); err != nil {
		t.Fatalf("SetBudget(%+v): %v", b, err)
	}
}

func TestBudgetWithinLimit(t *testing.T) {
	reset()
	mustSetBudget(t, Budget{Category: "Еда", Limit: 5000})

	if err := AddTransaction(Transaction{Amount: 3000, Category: "Еда"}); err != nil {
		t.Fatalf("ожидался успех: %v", err)
	}
	// Ровно до лимита тоже можно.
	if err := AddTransaction(Transaction{Amount: 2000, Category: "еда"}); err != nil {
		t.Fatalf("транзакция ровно до лимита должна проходить: %v", err)
	}
	if got := SpentByCategory("Еда"); got != 5000 {
		t.Errorf("потрачено %.2f, ожидалось 5000", got)
	}
}

func TestBudgetFloatPrecision(t *testing.T) {
	reset()
	mustSetBudget(t, Budget{Category: "Еда", Limit: 0.3})
	_ = AddTransaction(Transaction{Amount: 0.1, Category: "Еда"})

	// 0.1 + 0.2 = 0.30000000000000004 во float64, но по копейкам это ровно лимит.
	if err := AddTransaction(Transaction{Amount: 0.2, Category: "Еда"}); err != nil {
		t.Errorf("ожидался успех: %v", err)
	}
	if err := AddTransaction(Transaction{Amount: 0.01, Category: "Еда"}); !errors.Is(err, ErrBudgetExceeded) {
		t.Errorf("ожидалась ErrBudgetExceeded, получено %v", err)
	}
}

func TestBudgetExceeded(t *testing.T) {
	reset()
	mustSetBudget(t, Budget{Category: "Еда", Limit: 5000})
	_ = AddTransaction(Transaction{Amount: 4500, Category: "Еда"})

	err := AddTransaction(Transaction{Amount: 600, Category: "Еда"})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("ожидалась ErrBudgetExceeded, получено %v", err)
	}
	if n := len(ListTransactions()); n != 1 {
		t.Errorf("транзакция сверх бюджета не должна сохраняться, в хранилище %d", n)
	}
}

func TestBudgetCategoryCaseInsensitive(t *testing.T) {
	reset()
	mustSetBudget(t, Budget{Category: "еда", Limit: 100})

	if err := AddTransaction(Transaction{Amount: 150, Category: " ЕДА "}); !errors.Is(err, ErrBudgetExceeded) {
		t.Errorf("бюджет должен применяться без учёта регистра и пробелов, получено %v", err)
	}
}

func TestNoBudgetNoLimit(t *testing.T) {
	reset()
	mustSetBudget(t, Budget{Category: "Еда", Limit: 10})

	if err := AddTransaction(Transaction{Amount: 1_000_000, Category: "Путешествия"}); err != nil {
		t.Errorf("категория без бюджета не ограничена: %v", err)
	}
}

func TestSetBudgetUpdatesLimit(t *testing.T) {
	reset()
	mustSetBudget(t, Budget{Category: "Транспорт", Limit: 100})
	tx := Transaction{Amount: 300, Category: "Транспорт"}

	if err := AddTransaction(tx); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("ожидалась ErrBudgetExceeded, получено %v", err)
	}
	mustSetBudget(t, Budget{Category: "транспорт", Limit: 500})
	if err := AddTransaction(tx); err != nil {
		t.Fatalf("после увеличения лимита ожидался успех: %v", err)
	}
	if b, _ := GetBudget("Транспорт"); b.Limit != 500 {
		t.Errorf("лимит %.2f, ожидалось 500", b.Limit)
	}
}

func TestSetBudgetValidation(t *testing.T) {
	reset()
	if err := SetBudget(Budget{Category: "", Limit: 100}); !errors.Is(err, ErrEmptyCategory) {
		t.Errorf("ожидалась ErrEmptyCategory, получено %v", err)
	}
	if err := SetBudget(Budget{Category: "Еда", Limit: 0}); !errors.Is(err, ErrInvalidLimit) {
		t.Errorf("ожидалась ErrInvalidLimit, получено %v", err)
	}
	if err := SetBudget(Budget{Category: "Еда", Limit: -5}); !errors.Is(err, ErrInvalidLimit) {
		t.Errorf("ожидалась ErrInvalidLimit, получено %v", err)
	}
}

func TestBudgetConcurrent(t *testing.T) {
	reset()
	mustSetBudget(t, Budget{Category: "Еда", Limit: 1000})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = AddTransaction(Transaction{Amount: 50, Category: "Еда"})
		}()
	}
	wg.Wait()

	if got := SpentByCategory("Еда"); got != 1000 {
		t.Errorf("при параллельном добавлении потрачено %.2f, ожидалось ровно 1000", got)
	}
}

func TestLoadBudgets(t *testing.T) {
	reset()
	input := `[
		{"category": "Еда", "limit": 5000},
		{"category": "Транспорт", "limit": 3000.5}
	]`
	if err := LoadBudgets(strings.NewReader(input)); err != nil {
		t.Fatalf("LoadBudgets: %v", err)
	}
	if b, ok := GetBudget("еда"); !ok || b.Limit != 5000 {
		t.Errorf("бюджет «Еда»: %+v, ok=%v", b, ok)
	}
	if b, ok := GetBudget("Транспорт"); !ok || b.Limit != 3000.5 {
		t.Errorf("бюджет «Транспорт»: %+v, ok=%v", b, ok)
	}
}

func TestLoadBudgetsErrors(t *testing.T) {
	cases := []struct {
		name, input, wantSubstr string
	}{
		{"пустой ввод", "", "пустые входные данные"},
		{"синтаксис", `[{"category": "Еда", "limit": 5000,}]`, "синтаксическая ошибка"},
		{"обрыв", `[{"category": "Еда"`, "обрывается"},
		{"не массив", `{"category": "Еда", "limit": 5000}`, "ожидался JSON-массив"},
		{"null", `null`, "получено null"},
		{"тип поля", `[{"category": "Еда", "limit": "много"}]`, `поле "limit"`},
		{"неизвестное поле", `[{"category": "Еда", "limt": 5000}]`, "limt"},
		{"нулевой лимит", `[{"category": "Еда", "limit": 0}]`, "элемент #1"},
		{"пустая категория", `[{"category": "Еда", "limit": 1}, {"category": "", "limit": 1}]`, "элемент #2"},
		{"лишние данные", `[] []`, "лишние данные"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reset()
			err := LoadBudgets(strings.NewReader(c.input))
			if err == nil {
				t.Fatalf("ожидалась ошибка")
			}
			if !strings.Contains(err.Error(), c.wantSubstr) {
				t.Errorf("ошибка %q не содержит %q", err, c.wantSubstr)
			}
		})
	}
}

func TestLoadBudgetsAtomic(t *testing.T) {
	reset()
	input := `[{"category": "Еда", "limit": 5000}, {"category": "Транспорт", "limit": -1}]`
	if err := LoadBudgets(strings.NewReader(input)); err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if _, ok := GetBudget("Еда"); ok {
		t.Error("при ошибке в одном элементе бюджеты не должны применяться частично")
	}
}
