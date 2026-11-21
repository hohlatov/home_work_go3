# Домашнее задание 2 — Gateway и Ledger

Два независимых Go-сервиса в одном репозитории, у каждого свой `go.mod`.

```
.
├── gateway/                  # HTTP-шлюз
│   ├── go.mod
│   └── main.go               # HTTP-сервер на :8080, обработчик GET /ping
└── ledger/                   # бизнес-логика
    ├── go.mod
    ├── main.go               # запуск сервиса и демонстрация работы с транзакциями
    └── internal/ledger/
        ├── ledger.go         # Transaction, хранилище в памяти, AddTransaction, ListTransactions
        └── ledger_test.go    # unit-тесты
```

Нужен Go 1.22 или новее.

## Задание 1. Gateway

```bash
cd gateway
go run .
```

В другом терминале:

```bash
curl -i http://localhost:8080/ping
```

Ожидаемый ответ:

```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8

pong
```

Другие методы (например, `POST /ping`) получают `405 Method Not Allowed`.
Остановить сервер: `Ctrl+C`.

## Задание 2. Ledger

```bash
cd ledger
go run .
```

Ожидаемый вывод:

```
Ledger service started
... не удалось добавить транзакцию "Некорректная транзакция с нулевой суммой": сумма транзакции не может быть равна 0

Список транзакций (3):
ID   Дата         Категория         Сумма  Описание
1    2026-11-18   Продукты        1250.50  Покупки в супермаркете
2    2026-11-19   Транспорт        350.00  Такси до университета
3    2026-11-20   Образование     4990.00  Подписка на курс по Go
```

Как это устроено (`ledger/internal/ledger/ledger.go`):

- `Transaction` — структура с полями `ID`, `Amount`, `Category`, `Description`, `Date` (`time.Time`).
- Хранилище — срез `[]Transaction` в памяти, при запуске он пустой. Доступ защищён `sync.RWMutex`.
- `AddTransaction(tx Transaction) error` — назначает `ID = len(slice)+1` и добавляет транзакцию.
  Возвращает ошибку, если сумма равна 0 или категория пустая. Если дата не указана, подставляет текущее время.
- `ListTransactions() []Transaction` — возвращает копию всех транзакций.

В `main` добавляются три корректные транзакции и одна с нулевой суммой. Последняя отклоняется,
чтобы показать, как работает валидация.

Запуск тестов:

```bash
cd ledger
go test ./...
```
