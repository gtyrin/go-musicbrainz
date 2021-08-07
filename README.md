# go-musicbrainz #

Микросервис-клиент [Musicbrainz](https://musicbrainz.org/doc/MusicBrainz_API).
Обмен сообщениями с микросервисом реализован с использованием [RabbitMQ](https://www.rabbitmq.com).

Команды микросервиса:
---
|Команда|                    Назначение                      |
|-------|----------------------------------------------------|
|release|поиск по неполным метаданным или ID в БД Musicbrainz|
|ping   |проверка жизнеспособности микросервиса              |

*Пример использования команд приведен в тестовом клиенте в [musicbrainz.py](https://github.com/ytsiuryn/ds-musicbrainz/blob/main/musicbrainz.py)*.

Системные переменные для проведения тестов:
---
|    Переменная    |                              Назначение                           |
|------------------|-------------------------------------------------------------------|
|MUSICBRAINZ_APP   |наименование зарегистрированного приложения                        |
|MUSICBRAINZ_KEY   |                                                                   |
|MUSICBRAINZ_SECRET|секретный код клиента, полученный в ходе регистрации на Musicbrainz|


Пример запуска микросервиса:
---
```go
    package main

    import (
	    "flag"
	    "fmt"

	    log "github.com/sirupsen/logrus"

	    musicbrainz "github.com/ytsiuryn/ds-musicbrainz"
	    srv "github.com/ytsiuryn/ds-service"
    )

    func main() {
		connstr := flag.String(
			"msg-server",
			"amqp://guest:guest@localhost:5672/",
			"Message server connection string")

		product := flag.Bool(
			"product",
			false,
			"product-режим запуска сервиса")

		flag.Parse()

		log.Info("Start ", musicbrainz.ServiceName)

		cl := musicbrainz.New(
			os.Getenv("MUSICBRAINZ_APP"),
			os.Getenv("MUSICBRAINZ_KEY"),
			os.Getenv("MUSICBRAINZ_SECRET"))

		msgs := cl.ConnectToMessageBroker(*connstr)

		if *product {
			cl.Log.SetLevel(log.InfoLevel)
		} else {
			cl.Log.SetLevel(log.DebugLevel)
		}

		cl.Start(msgs)
    }
```

Пример клиента (Python тест)
---
См. файл [musicbrainz.py](https://github.com/ytsiuryn/ds-musicbrainz/blob/main/musicbrainz.py)