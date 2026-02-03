# Нормализация поисковых запросов, часть 2

## Цель

Создать микросервис, который будет REST-шлюзом для поисковых микросервисов.

Mикросервис API gateway должен работать по REST протоколу в соответсвии с предложенной
схемой:

1. По запросу "GET /ping" сервис отдает ответ о состоянии поисковых сервисов.
При таком запросе gateway запрашивает по gRPC состояния других микросервисов.
Ответ должен быть в виде JSON:

    ```json
    {
        "replies": [
            "words": "ok",
            "other service": "unavailable"
        ]
    }
    ```

    В текущем задании у нас только один внутренний сервис с именем "words".

2. По запросу "GET /api/words?phrase="follow, followers" cервис должен принимать на вход строку
(на английском) и возвращать назад нормализованный вид в виде структуры со слайсом слов.
То есть при посылке "follower brings bunch of questions" сервер должен отдать
["follow", "bring", "bunch", "question"] - слова в слайсе в любом порядке. Формат ответа - JSON:

    ```json
    {
        "words": ["follow", "bring", "bunch", "question"],
        "total": 4
    }
    ```

    При успешном запросе отдается HTTP статус OK, при отсутствии или пустом phrase - Bad Request.
    Также нужно реализовать трансляцию gRPC кода при получении сообщения больше 4 KiB -
    ResourceExhausted в Bad Request.

Вам дана структура проекта без имплементации в виде популярной архитектуры Ports & Adapters,
она же гексагональная архитектура. Такая архитектура предполагает принцип инверсии зависимостей.
Попробуйте этот принцип реализовать.

Сервис должен собираться и запускаться через предоставленный compose файл,
а также проходить интеграционные тесты - запуск специального тест контейнера.

## Критерии приемки

1. Микросервис компилируeтся в docker image, запускаeтся через compose файл и проходит тесты.
2. Используется микросервис words из предыдущего задания.
3. Сервер конфигурируeтся через cleanenv пакет и должeн уметь запускаться как с config.yaml файлом
через флаг -config, так и через переменные среды, в этом задании - HTTP_SERVER_ADDRESS,
HTTP_SERVER_TIMEOUT, WORDS_ADDRESS, LOG_LEVEL
4. Используется golang 1.24+, slog логгер.

## Материалы для ознакомления

- [Как создавать модули](https://go.dev/doc/tutorial/create-module)
- [Учимся разрабатывать REST API на Go на примере сокращателя ссылок](https://habr.com/ru/companies/selectel/articles/747738/)
- [Порты и адаптеры, Wikipedia](https://ru.wikipedia.org/wiki/%D0%93%D0%B5%D0%BA%D1%81%D0%B0%D0%B3%D0%BE%D0%BD%D0%B0%D0%BB%D1%8C%D0%BD%D0%B0%D1%8F_%D0%B0%D1%80%D1%85%D0%B8%D1%82%D0%B5%D0%BA%D1%82%D1%83%D1%80%D0%B0)
- [Go Clean Template](https://www.youtube.com/watch?v=V6lQG6d5LgU)
- [SOLID в Go](https://habr.com/ru/companies/domclick/articles/816885/)
- [gRPC Quick Start](https://grpc.io/docs/languages/go/quickstart/)
- [gRPC Basic Tutorial](https://grpc.io/docs/languages/go/basics/)
