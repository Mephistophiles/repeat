# Архитектура `repeat`

## CLI

```
repeat [flags] <N> <command> [args...]
```

### Флаги

| Флаг | Назначение |
|------|-----------|
| `-v`, `--verbose` | Вывод команды: без TUI — в stdout, в TUI — в скроллящейся области |
| `-d`, `--delay` | Задержка между запусками (`1s`, `500ms` ...) |
| `-t`, `--timeout` | Таймаут на один запуск команды (`30s`, `5m` ...). Без флага — без таймаута |
| `--continue` | Выполнить все N раз, не останавливаясь на ошибках |
| `--until-success` | Бесконечно, пока команда не вернёт 0 (несовместим с N) |
| `--progress` | Bubbletea TUI с прогресс-баром, ETA, статусом, выводом (если `-v`) |
| `--json` | JSON-сводка по итогам всех запусков (после TUI, если `--progress`) |
| `-h`, `--help` | Справка |
| `--version` | Версия |

### Несовместимые комбинации

- `--until-success` + `<N>` -> ошибка, выбирай что-то одно
- `--continue` игнорируется при `--until-success`

---

## Структура проекта

```
repeat/
├── cmd/repeat/
│   └── main.go              # Точка входа, парсинг флагов, валидация, оркестрация
├── internal/
│   ├── runner/
│   │   ├── runner.go        # exec.CommandContext, захват stdout/stderr, таймауты
│   │   └── result.go        # Result struct: exit code, duration, raw output
│   ├── log/
│   │   └── log.go           # Сессия логов: директория, symlink last, запись run.N.log
│   ├── tui/
│   │   ├── model.go         # Bubbletea Model: прогресс, ETA, состояние
│   │   ├── update.go        # Update: обработка сообщений от runner
│   │   └── view.go          # View: рендер прогресс-бара, ETA, текущий запуск
│   └── summary/
│       └── summary.go       # JSON-структура финальной сводки
├── test/
│   └── integration/
│       └── integration_test.go  # Интеграционные тесты (build tag)
├── go.mod
└── go.sum
```

### Зависимости

- `charmbracelet/bubbletea`
- `charmbracelet/bubbles` (прогресс-бар)
- Стандартная библиотека Go

---

## Потоки выполнения

### 1. Базовый режим: `repeat 3 echo hello`

```
parseFlags() -> createSession() -> for i := 1; i <= N; i++:
  runner.Run(cmd, args) -> log.writeRun(i, result) -> if !result.OK -> os.Exit(result.Code)
// success -> os.Exit(0)
```

### 2. Режим `--continue`: `repeat --continue 3 sh -c 'exit 1'`

```
// То же, но не выходим при ошибке.
// В конце выводим сводку (кол-во успехов/ошибок), exit code = код последней ошибки или 0
```

### 3. Режим `--until-success`: `repeat --until-success curl https://api`

```
for i := 1; ; i++:
  runner.Run(cmd, args) -> if result.OK -> os.Exit(0)
  sleep(delay)
```

### 4. Режим `--progress` (Bubbletea TUI)

Bubbletea работает в event-loop (Model → Update → View). Запуск команд — блокирующая операция, поэтому выполнение вынесено в горутину, которая отправляет сообщения (`RunStarted`, `RunCompleted`, `RunOutput`) через канал в Update-функцию.

#### 4a. Обычный режим: `repeat --progress 10 echo hello`

```
  ┌──────────────────────────────────┐
  │ repeat: echo hello               │
  │                                  │
  │ [=========>         ] 3/10  30%  │
  │                                  │
  │ ETA: ~2m 15s (avg 22s/run)       │
  │                                  │
  │ > Run 3: completed (exit 0, 22ms)│
  │ > Run 4: running...              │
  └──────────────────────────────────┘
```

#### 4b. С verbose: `repeat --progress -v 10 pytest`

```
  ┌──────────────────────────────────┐
  │ repeat: pytest test_foo.py       │
  │                                  │
  │ [=========>         ] 3/10  30%  │
  │                                  │
  │ ETA: ~2m 15s (avg 22s/run)       │
  │                                  │
  │ > Run 3: completed (exit 0, 22ms)│
  │ > Run 4: running...              │
  │──────────────────────────────────│
  │ tests/test_foo.py::test_bar PASSED│
  │ tests/test_foo.py::test_baz FAILED│
  │ ...                              │
  └──────────────────────────────────┘
```

Скроллящаяся область в нижней части TUI показывает вывод текущей команды в реальном времени.

#### 4c. `--until-success` + `--progress`: `repeat --until-success --progress curl api`

```
  ┌──────────────────────────────────┐
  │ repeat: curl https://api         │
  │                                  │
  │ ⠋ Attempt 42                     │
  │                                  │
  │ Elapsed: 3m 22s                  │
  │                                  │
  │ Last: fail (exit 7, 120ms)       │
  └──────────────────────────────────┘
```

Вместо прогресс-бара — спиннер (`⠋⠙⠹⠸...`) и счётчик попыток + elapsed time.

#### Механика Bubbletea

```
main горутина:                      runner горутина:
  tea.NewProgram(model)              │
  └─ Init() → RunNextCmd             └─ for each run:
       │                                  runner.Run(cmd)
       ├─ Update(msg) ←────────────────── chan→ RunStarted
       │       │                           │
       ├─ Update(msg) ←────────────────── chan→ RunOutput (стриминг при -v)
       │       │                           │
       ├─ Update(msg) ←────────────────── chan→ RunCompleted
       │       │
       └─ View() → рендер
```

Ключевые сообщения:
- `RunStarted{RunIndex}` — начался новый запуск
- `RunOutput{RunIndex, Line}` — строка вывода (только при `-v`, для скроллящейся области)
- `RunCompleted{Result}` — запуск завершён
- `Interrupted` — пользователь нажал Ctrl+C

**ETA-формула:** `avg_duration × remaining_runs`. Показывается с первого запуска, корректируется после каждого следующего. При `--delay` учитывается в ETA: `avg_duration × remaining + delay × remaining`.

---

## Структура Result

```go
type Result struct {
    RunIndex    int
    Command     string
    ExitCode    int
    StartedAt   time.Time
    FinishedAt  time.Time
    Duration    time.Duration
    Stdout      string // обрезается до 64KB
    Stderr      string // обрезается до 64KB
    Interrupted bool   // true если был SIGINT
    TimedOut    bool   // true если превышен --timeout
}
```

**Ограничение буфера:** stdout и stderr хранятся в памяти с лимитом 64KB каждый. Полный вывод всегда пишется в лог-файл без ограничений. При обрезании в лог добавляется пометка `[output truncated at 64KB]`.

---

## Формат лог-файла `run.N.log`

```
# repeat — run 3 of 5
# command: echo hello
# started:  2026-06-07T14:30:05.123+03:00
# finished: 2026-06-07T14:30:05.145+03:00
# duration: 22ms
# exit code: 0
---
hello
```

Всё в одном файле: метаданные в виде комментариев (`#`), затем разделитель `---`, затем вывод команды.

---

## Хранение логов

- Директория: `$PWD/.repeat/<ISO8601_TIMESTAMP>/` (например `.repeat/2026-06-07T14-30-05/`)
- Файлы: `run.1.log`, `run.2.log`, ..., `run.N.log`
- Symlink: `.repeat/last` -> последняя директория сессии

---

## JSON-сводка (`--json`)

```json
{
  "command": "echo hello",
  "runs": [
    {"run":1,"exit_code":0,"duration_ms":22,"stdout":"hello","stderr":""},
    {"run":2,"exit_code":0,"duration_ms":18,"stdout":"hello","stderr":""}
  ],
  "total": 2,
  "successes": 2,
  "failures": 0,
  "total_duration_ms": 40
}
```

Выводится в stdout после завершения всех запусков. При `--progress` — после выхода из TUI и восстановления терминала.

---

## Обработка сигналов (SIGINT)

1. Послать SIGINT дочернему процессу
2. Дождаться его завершения
3. Записать лог с `interrupted: true`
4. Обновить symlink `last`
5. Если `--progress` — закрыть TUI, восстановить терминал
6. `os.Exit(130)`

---

## План тестирования

| Тип | Что тестируем | Инструменты |
|-----|--------------|-------------|
| **Unit** `runner` | Запуск команды, захват stdout/stderr, exit code, таймаут | `os/exec`, моки `io.Writer` |
| **Unit** `log` | Создание сессии, запись файлов, symlink `last` | `t.TempDir()` |
| **Unit** `summary` | JSON-сериализация `[]Result` | `encoding/json` |
| **Unit** `tui` | ETA-калькуляция, форматирование прогресс-бара | Чистые функции без Bubbletea |
| **Integration** | Полный цикл: `repeat 3 echo hello` в реальном процессе | `os/exec` + `t.TempDir()`, проверка файловой системы |
| **Integration** | Флаги: `--continue`, `--delay`, `--json`, `--until-success` | Табличные тесты |
| **Integration** | SIGINT-обработка | `Process.Signal()` |
| **Integration** | Невалидные аргументы, exit codes | `TestMain` / subprocess |

Интеграционные тесты — отдельный пакет с build tag `integration` (не замедляют `go test ./...`).

---

## Что НЕ входит в v1

- Конфиг-файл `.repeat.yml`
- Параллельное выполнение
- Раздельный stdout/stderr в разных файлах
- Subcommands (`repeat log`, `repeat list`, `repeat clean`)
- Фильтрация exit codes (`--skip-exit`, `--exit-on`)
- Защита от race condition на symlink `.repeat/last` при параллельных запусках `repeat`
- Парсинг лог-файлов машиной (формат рассчитан на чтение человеком)

## Известные ограничения

- **ETA по первому запуску:** первая оценка основана на 1 точке — может сильно отклоняться, корректируется с каждым следующим запуском
- **Коллизия формата лога:** если команда выводит строки `---` или `# comment`, они неотличимы от метаданных. Формат рассчитан на чтение человеком, не на машинный парсинг
- **`-v` + `--progress`:** вывод команды идёт только в скроллящуюся область TUI, не в stdout напрямую. При выходе из TUI вывод команды остаётся только в лог-файлах
