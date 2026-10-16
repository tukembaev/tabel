# Overall Info
Tabel — микросервис электронного табеля КГТУ: сотрудники (включая преподавателей) отмечают приход и уход сканированием QR у входа, система считает отработанное время, принимает заявки на неявки и отдаёт отчёты заведующим.

Как работает
Логин через uadmin → JWT (Bearer, HS256, sub = UUID).
Клиент один раз за сессию (~раз в сутки) делает POST /v1/me/sync с данными из /users/me — бек хранит snapshot профиля сотрудника.
Дальше все операции идут только в Tabel по JWT; user_id из токена, не из body.
Гео (на территории кампуса) — на клиенте; бек парсит QR, валидирует сотрудника, дубли и cutoff (18:30).
Фоновый worker (Redis/asynq): автозакрытие открытых смен в 18:30, архивация прошлых семестров.

Основные API
Группа	Эндпоинты
Sync
POST /v1/me/sync
Отметка
POST /v1/attendance/scan
Табель зав
PUT /timesheet/days, GET /timesheet/edits
Отдел
GET /departments/{id}/employees, /export, /missing
Неявки
POST /absences/requests, GET .../pending, POST .../head, POST .../{id}/decide
Отчёты
GET /reports/me, /department, /departments/{id}, /users/{id} (?from=&to=)
Health
GET /healthz
Swagger: /swagger/.

Стек
Go + Chi, Postgres (pgx, sqlc, goose), Redis + asynq (воркер), JWT HS256, slog. Прод — бинарники + systemd (tabel-api, tabel-worker), Docker только локально для Postgres/Redis.

# Manual для Frontend / Mobile разработчика — Tabel API

Сервис табеля КГТУ. Этот документ — контракт для связки мобильного (и web) клиента с беком.

- **Swagger UI:** `GET http://<host>:8080/swagger/`
- **OpenAPI:** `GET http://<host>:8080/swagger/openapi.yaml` (также файл [`api/openapi.yaml`](../api/openapi.yaml))
- **Health:** `GET /healthz` (без токена)

---

## 1. Базовые правила

| Правило | Детали |
|--------|--------|
| Base URL | Пример prod: `https://tabel.kstu.kg` / local: `http://localhost:8080` |
| Format | JSON, `Content-Type: application/json` |
| Auth | `Authorization: Bearer <access_token>` на все `/v1/*` |
| JWT | Тот же access-токен, что от uadmin (HS256). В токене нужен `sub` = UUID пользователя |
| User id | **Никогда** не передавать `user_id` в body для «себя» — бек берёт из JWT |
| Ошибки | `{ "error": "текст" }` + HTTP-код |

### HTTP-коды

| Code | Когда |
|------|--------|
| 200 / 201 | OK / создан новый цикл (check_in) |
| 400 | Невалидный QR, check-in после 18:30 |
| 401 | Нет/битый JWT |
| 403 | Нет прав (не сотрудник, не зав, head-only код) |
| 404 | Нет ресурса |
| 409 | Конфликт (approved absence / заявка при уже существующей отметке) |
| 428 | Нет snapshot — сначала `POST /v1/me/sync` |

---

## 2. Обязательный flow после логина

```
1) Login → access JWT (как сейчас в unet/uadmin)
2) GET https://uadmin.kstu.kg/users/api/v1/users/me
3) POST {tabel}/v1/me/sync   ← один раз на сессию приложения (~раз в 24ч)
4) Дальше сканы / отчёты только в tabel (+ JWT), без /users/me на каждый QR
```

Клиент хранит `lastSyncAt`. Если сессия новая или прошло > 24ч — снова sync.

Доступ только у сотрудников: `has_employee=true`. Студенты без employee-профиля получат 403.

### `POST /v1/me/sync`

**Headers:** `Authorization: Bearer …`

**Request:**

```json
{
  "username": "20809200450206",
  "first_name": "Арслан",
  "last_name": "Жумалиев",
  "middle_name": "",
  "has_employee": true,
  "is_department_head": false,
  "can_edit_timesheet": false,
  "department_id": "id-отдела-из-employee_profile",
  "department_name": "Кафедра …"
}
```

Маппинг из `/users/me`:

| Поле sync | Откуда |
|-----------|--------|
| username, first_name, last_name, middle_name | корень ответа |
| has_employee | `employee_profile.is_active == true` |
| is_department_head | по permission / роли зава (клиент решает) |
| can_edit_timesheet | true для зава, кадровика, заместителя — кто может править табель отдела |
| department_id / department_name | из `employee_profile.employments[0]` (или актуальный employment) |

**Response `200`:** объект snapshot (см. схему UserSnapshot в Swagger).

---

## 3. QR

Физический стикер на входе в корпус/кабинет кодирует URL:

```
https://qr.kstu.kg/kstu/1/102
```

| Сегмент | Смысл | Пример |
|---------|--------|--------|
| host | всегда `qr.kstu.kg` | |
| `/kstu` | university_id | должен совпасть с `UNIVERSITY_ID` бека |
| `/1` | корпус (`entry_building` / `exit_building`) | `1` |
| `/102` | кабинет (`entry_room` / `exit_room`) | `102` |

Клиент шлёт **весь URL как есть** в `qr_url`. Парсинг и валидация — на беке. Не собирать `building`/`room` руками на клиенте.

Рекомендуемые проверки **на клиенте** перед запросом:

- Гео внутри кампуса.
- Пользователь — сотрудник (`has_employee`).
- После успешного скана — disable кнопки на 2–3 секунды (UX; сервер всё равно защищён окном дублей 2 минуты).

---

## 4. Единая точка отметки

### `POST /v1/attendance/scan`

```json
{ "qr_url": "https://qr.kstu.kg/kstu/1/102" }
```

Бек сам решает, что это: приход, уход или случайный повтор. **Один цикл в день:** первый скан = приход, второй = уход. Обед не сканируется — час (12:00–13:00) вычитается автоматически, если человек был на работе в это окно.

**Response `201`** (новый приход) / **`200`** (уход, игнор дубля, день уже закрыт):

```json
{
  "action": "check_in",
  "session": {
    "id": "session-uuid",
    "user_id": "…",
    "work_date": "2026-08-07",
    "status": "open",
    "entry_building": "1",
    "entry_room": "102",
    "exit_building": null,
    "exit_room": null,
    "opened_at": "2026-08-07T08:00:00Z",
    "closed_at": null,
    "close_reason": null,
    "duration_seconds": null,
    "display_name": "Жумалиев Арслан"
  }
}
```

| `action` | HTTP | Смысл для UI |
|----------|------|----------------|
| `check_in` | 201 | Приход. Toast: «Приход зафиксирован» |
| `check_out` | 200 | Уход. Toast: «Уход зафиксирован». Если раньше стояло авто 18:30 — время заменяется на этот скан |
| `duplicate_ignored` | 200 | Скан слишком близко к предыдущему (< 2 мин). Toast: «Уже отмечены» |
| `already_closed` | 200 | День уже закрыт вручную. Toast: «Уход уже зафиксирован» |

### Сценарии (в том числе «забыл QR»)

| Ситуация | Что делает бек |
|----------|----------------|
| Первый скан за день в 08:00 / 09:00 | `check_in` |
| Повтор через 30 секунд | `duplicate_ignored` |
| Второй скан в любой момент (в т.ч. 21:00) | `check_out`. Обед 12:00–13:00 вычтен сам, если интервал его накрывает |
| Обед | **не сканировать.** Не открывается второй цикл. В отчёте `lunch_seconds` |
| Забыл уход | Воркер в **18:30** закроет смену (`auto_cutoff`). Это **не прогул**, человек был |
| Пришёл в 9:00, ушёл в 21:00, авто уже закрыло в 18:30 | Второй скан **заменяет** 18:30 на 21:00 (`close_reason: manual`) |
| Забыл оба скана | День `kind=missing`, `needs_review=true`. **Не прогул.** Зав ставит явку через `PUT /v1/timesheet/days` |
| Был, но не умеет сканировать | Зав: `kind=attendance` без времени → 08:00–18:30 минус обед |
| Отпуск/больничный | Зав ставит код. Отметиться QR нельзя (409) |
| Первый скан после 18:30 без прихода | `400` — приход уже нельзя; зав правит день вручную |
| Не сотрудник / битый QR | 403 / 400 |

Прогул (`П`) **никогда не ставится автоматически**. Только зав/редактор табеля.

Cutoff 18:30 закрывает забытый уход, но живой второй скан всегда важнее автозакрытия.

---

## 5. Неявки / коды табеля

### Сотрудник: заявка

`POST /v1/absences/requests`

```json
{
  "work_date": "2026-08-06",
  "code": "БЛ",
  "note": "Больничный"
}
```

Дата: **`YYYY-MM-DD`**. Нельзя, если в этот день уже была хотя бы одна отметка → 409.

Статус созданной: `pending`.

### Зав: список pending

`GET /v1/absences/pending` — нужен `is_department_head` или `can_edit_timesheet`.

### Зав: approve/reject

`POST /v1/absences/{id}/decide`

```json
{ "approve": true }
```

### Зав: сразу поставить код

`POST /v1/absences/head`

```json
{
  "user_id": "15b7fbe4-95d0-4319-a6c5-fa1ea13391ed",
  "work_date": "2026-08-06",
  "code": "П",
  "note": "Прогул"
}
```

Только сотрудники **своего** `department_id`, и только без отметок в этот день.

### Справочник кодов (для UI)

| Код | Кто может | Примечание |
|-----|-----------|------------|
| Б, БЛ, Больничный | сотрудник заявка / зав | |
| О, Отпуск | сотрудник / зав | |
| ОБС, Д, Ж, Т, учебные, командировка | сотрудник / зав | |
| П, Прогул | **только зав** | не автоматом |
| Ув, Уволен | **только зав** | |
| ПР, Праздник | зав | |
| сб / Воскресение | система в отчётах | не заявка |

---

## 5.1. Правка табеля завом / редактором

Кто: `is_department_head` или `can_edit_timesheet`, только свой `department_id`.

### `PUT /v1/timesheet/days`

Человек был на работе, но не отсканировал:

```json
{
  "user_id": "15b7fbe4-95d0-4319-a6c5-fa1ea13391ed",
  "work_date": "2026-09-17",
  "kind": "attendance",
  "note": "был, QR не сканировал"
}
```

Без `opened_at`/`closed_at` ставится стандартный день 08:00–18:30 минус обед.

Со своими часами:

```json
{
  "user_id": "15b7fbe4-95d0-4319-a6c5-fa1ea13391ed",
  "work_date": "2026-09-17",
  "kind": "attendance",
  "opened_at": "2026-09-17T09:00:00+06:00",
  "closed_at": "2026-09-17T17:00:00+06:00",
  "note": "по журналу кафедры"
}
```

Неявка:

```json
{ "user_id": "…", "work_date": "2026-09-17", "kind": "absence", "code": "О", "note": "отпуск" }
```

Сбросить день:

```json
{ "user_id": "…", "work_date": "2026-09-17", "kind": "clear" }
```

Журнал правок: `GET /v1/timesheet/edits?from=&to=`.

---

## 6. Отчёты (mobile + web)

Query: `?from=2026-08-01&to=2026-08-31` (если нет — текущий месяц).

| Endpoint | Кто |
|----------|-----|
| `GET /v1/reports/me` | любой сотрудник |
| `GET /v1/reports/users/{userID}` | сам или зав/редактор того же отдела |
| `GET /v1/reports/department` | зав/редактор — **все сотрудники отдела**, включая дни без QR (`needs_review`) |
| `GET /v1/reports/departments/{departmentID}` | то же, явно по id отдела |
| `GET /v1/departments/{departmentID}/employees` | список snapshot отдела |
| `GET /v1/departments/{departmentID}/missing?from=&to=` | дни без отметки (для разбора с людьми, которые не сканируют) |
| `GET /v1/departments/{departmentID}/export?from=&to=` | полный срез: сотрудники, сессии, неявки, правки, табель, missing |

`kind=missing` + `needs_review=true` — это очередь зава, не прогул.

`total_seconds` — сумма **отработанных** часов (`duration − lunch`).

---

## 7. Модель AttendanceSession

| Поле | Тип | Значения |
|------|-----|----------|
| id | uuid | |
| status | string | open / closed |
| close_reason | string\|null | manual / auto_cutoff / head |
| source | string | scan / head / system |
| work_date | date | день учёта |
| opened_at / closed_at | datetime | RFC3339 |
| duration_seconds | int\|null | «грязь» от входа до выхода |
| lunch_seconds | int | автообед (обычно 3600, если день накрыл 12:00–13:00) |
| entry_building / entry_room | string | QR прихода |
| exit_building / exit_room | string\|null | QR ухода (null при auto_cutoff, пока не пересканировали) |
| edited_by / note | | кто правил день вручную |
| display_name | string | ФИО snapshot |

---

## 8. Рекомендуемые экраны

1. **Сотрудник:** скан QR → toast по `action`. Не спрашивать про обед.
2. **Мой табель:** `reports/me`.
3. **Зав — «Кто не отметился»:** `.../missing` за сегодня/неделю, кнопка «Был» → `PUT /timesheet/days` kind=attendance.
4. **Зав — отчёт отдела:** `.../export` или `reports/department`.
5. Неявки: pending + head codes.

---

## 9. Чеклист интеграции

- [ ] Bearer JWT на все `/v1/*`
- [ ] Sync после логина; поля `can_edit_timesheet` / `is_department_head`
- [ ] Полный `qr_url` в scan
- [ ] `action`: check_in / check_out / duplicate_ignored / already_closed
- [ ] Не строить обеденные циклы на клиенте
- [ ] Экран missing + правка дня для зава
- [ ] `needs_review` не показывать как прогул
- [ ] Даты absence `YYYY-MM-DD`
- [ ] 428 → sync; 409 отпуск; 400 after cutoff

---

## 10. Swagger

После запуска `tabel-server`:

```
http://localhost:8080/swagger/
```

В Authorize вставьте `Bearer <token>` (или только токен — в зависимости от UI; обычно без слова Bearer, UI добавит сам).
