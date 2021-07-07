# go-musicbrainz #

Микросервис-клиент [Musicbrainz](https://musicbrainz.org/doc/MusicBrainz_API).
Обмен сообщениями с микросервисом реализован с использованием [RabbitMQ](https://www.rabbitmq.com).

Команды микросервиса:
---
|Команда|                    Назначение                      |
|-------|----------------------------------------------------|
|release|поиск по неполным метаданным или ID в БД Musicbrainz|
|ping   |проверка жизнеспособности микросервиса              |
|info   |информация о микросервисе                           |

*Пример использования команд приведен в тестовом клиенте в [musicbrainz.py](https://github.com/ytsiuryn/ds-musicbrainz/blob/main/musicbrainz.py)*.

Окружение:
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
	    flag.Parse()

	    log.Info(fmt.Sprintf("%s starting..", musicbrainz.ServiceName))

	    cl, err := musicbrainz.NewMusicbrainzClient(*connstr)
	    srv.FailOnError(err, "Failed to create Musicbrainz client")

	    err = cl.TestPollingFrequency()
	    srv.FailOnError(err, "Failed to test polling frequency")

	    defer cl.Close()

	    cl.Dispatch(cl)
    }
```

Пример клиента (Python тест)
---
См. файл [musicbrainz.py](https://github.com/ytsiuryn/ds-musicbrainz/blob/main/musicbrainz.py)