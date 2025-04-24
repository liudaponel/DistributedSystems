# Распределенная информационная система по расшифровке MD5 хэша

## Описание проекта

Система представляет собой распределённое приложение для подбора исходных строк по их MD5-хэшам методом перебора *(bruteforce)*. Состоит из двух основных компонентов:
- Manager: координирует работу, распределяет задачи и обрабатывает запросы клиентов
- Worker: выполняет непосредственный перебор и поиск совпадений хэшей

## Запуск проекта

```bash
docker compose up --build -d
```

## API Endpoints

### Manager Public API

#### POST /api/hash/crack
Отправляет запрос на расшифровку хэша.

Request:
```json
{
    "hash": "098f6bcd4621d373cade4e832627b4f6",
    "maxLength": 4
}
```

Response:
```json
{
    "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### GET /api/hash/status?requestId={requestId}
Получает статус расшифровки по requestId.

Response:
```json
{
    "status": "DONE|IN_PROGRESS|FAIL",
    "data": ["найденная_строка"]
}
```

### Manager Internal API

#### POST /internal/api/manager/hash/crack/result
Внутренний endpoint для получения результатов от worker'ов.

Request:
```json
{
    "hash": "098f6bcd4621d373cade4e832627b4f6",
    "result": "test",
    "partNumber": 1
}
```

#### POST /internal/api/worker/register
Endpoint для регистрации новых worker'ов в системе.

Request:
```json
{
    "workerUrl": "http://worker:8081",
    "maxWorkers": 4
}
```

### Worker API

#### POST /internal/api/worker/hash/crack/task
Endpoint для получения заданий на расшифровку от manager'а.

Request:
```json
{
    "hash": "098f6bcd4621d373cade4e832627b4f6",
    "maxLength": 4,
    "partNumber": 1,
    "partCount": 8
}
```
