# Hello web server

## Цель

Создать два простых микросервиса, обслуживающих клиентов по REST протоколу, - hello и fileserver.

У hello сервиса должно быть 2 эндпоинта:

1. GET /ping - отвечаем HTTP OK и pong
2. GET /hello?name=Misha - отвечаем HTTP OK и Hello, Misha!

Fileserver должен идеоматично имплементировать CRUD для файлового хранилища:

1. POST /files - должен принимать файл в multipart форме и записывать его в определенную
пользователем папку. Ключ формы - "file". Возвращает статус "Created" и имя файла.
Если файл уже есть, возвращает статус "Conflict".
2. PUT /files/filename - заменяет содержимое файла в файловом хранилище. Имя файла записано в пути.
Файл должен существовать, иначе возвращает статус "NotFound".
3. GET /files - листинг файлов, отсортирован по именам, одно имя на строчку.
4. GET /files/filename - отдает точное содержание, если не найден - статус "NotFound"
5. DELETE /files/filename - идемпотентно удаляет файл.

Сервис должен собираться и запускаться через предоставленный compose файл,
а также проходить интеграционные тесты - запуск специального тест контейнера.

Полезные curl-ы:

```bash
curl -v -X POST -F file=@file1.txt localhost:28081/files
curl -v localhost:28081/files
curl -v localhost:28080/files/file.txt
curl -v -X DELETE localhost:28081/files/file1.txt
```

## Критерии приемки

1. Микросервисы компилируются в docker image, запускаются через compose файл и проходят тесты.
2. Используется только стандартная библиотека http.
3. Серверы конфигурируются через cleanenv пакет и должны уметь запускаться как с config.yaml
файлом через флаг -config, так и через переменные среды,
в этом задании - HELLO_PORT и FILESERVER_PORT.
4. Используется golang 1.25+

## Материалы для ознакомления

Git

- [git-for-half-an-hour](https://proglib.io/p/git-for-half-an-hour)
- [git-github-review](https://selectel.ru/blog/git-github-review/)

Make

- [makefiles-for-go-developers](https://tutorialedge.net/golang/makefiles-for-go-developers/)

Compose

- [gettingstarted](https://docs.docker.com/compose/gettingstarted/)

Project layout

- [flat-application-structure](https://www.calhoun.io/flat-application-structure/)

REST

- [rest-api](https://cloud.yandex.ru/ru/docs/glossary/rest-api)

Старый способ сделать HTTP server

- [rest_api_series](https://www.jetbrains.com/guide/go/tutorials/rest_api_series/stdlib/)

1.22 HTTP в Go 1.22+

- [spin-a-framework-free-http-router-server-in-go-122-with-ease-6nc](https://dev.to/prakash_chokalingam/spin-a-framework-free-http-router-server-in-go-122-with-ease-6nc)
- [routing-enhancements](https://go.dev/blog/routing-enhancements)
- [Улучшенная маршрутизация HTTP-серверов в Go 1.22](https://habr.com/ru/articles/768034/)
