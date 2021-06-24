# go-musicbrainz #

Микросервис-клиент [Musicbrainz](https://musicbrainz.org/doc/MusicBrainz_API).
Обмен сообщениями реализован с использованием [RabbitMQ](https://www.rabbitmq.com).

## Пример запуска микросервиса:
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

## Пример клиента Python (тест)

См. файл [musicbrainz.py](https://github.com/ytsiuryn/ds-musicbrainz/blob/main/musicbrainz.py)