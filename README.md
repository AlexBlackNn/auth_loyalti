# SSO  - Сервис авторизации

Единый сервис авторизации (СА) и управления пользователями 

## Swagger grpc доступен по адресу:
http://localhost:44044/sw/

## Swagger http доступен по адресу:
http://localhost:8000/swagger/index.html

## Архитектурные решения 

### API:
HTTP-handlers используются для общения с frontend, а gRPC для общения внутри микросервисов.
* gRPC: gRPC использует Protobuf для сериализации данных, что обеспечивает более компактное
представление сообщения по сравнению с JSON (используемым в HTTP), следовательно
реализуется более быстрый обмен данными.
* HTTP: HTTP, является стандартом для веб-разработки, что делает его простым и легким для
интеграции с frontend-фреймворками.


### Технический анализ особенностей регистрации пользователей

### 1. Немедленная выдача токенов

При регистрации пользователя система сразу выдает access и refresh токены, минуя этап входа в систему. Это позволяет пользователю начать взаимодействие с сервисом непосредственно после регистрации.

### 2. Использование Patroni кластера для хранения данных

Система использует Patroni кластер, обеспечивающий высокую доступность данных за счет репликации.

### 3. Асинхронная репликация и конечная согласованность

Репликация в Patroni кластере происходит асинхронно, что может привести к временной несогласованности между ведущим узлом и репликами. В результате, данные о новом пользователе могут отсутствовать на реплике в течение нескольких секунд после его регистрации.

### 4. Время жизни токенов и задержка репликации

Время жизни токенов установлено на 15 минут, что, как ожидается, будет достаточно для завершения репликации данных о пользователе на все реплики.

### 5. Временная несогласованность и ее влияние

Временная несогласованность данных на репликах, как правило, является кратковременным явлением. Она может привести к тому, что пользователь не сможет войти в систему сразу после регистрации, но это не должно быть проблемой, так как к моменту окончания срока действия токенов, репликация данных должна быть завершена.

### 6. Снижение нагрузки на БД за счет использования реплик

Чтение данных осуществляется с реплик, что значительно снижает нагрузку на ведущий узел. Это позволяет обеспечить более высокую производительность системы.

### 7. Возможные проблемы с задержкой репликации

Несмотря на то, что задержка репликации в штатном режиме невелика, при больших нагрузках или проблемах с сетью она может быть существенной. В этом случае пользователь может столкнуться с проблемами при авторизации. Вероятность такой ситуации мала, так как сильное отставание реплики в стабильно работающей системе встречается редко.

### 8. Компромисс между производительностью и согласованностью

Использование реплик для чтения данных представляет собой разумный компромисс между производительностью и согласованностью. Несмотря на риск временной несогласованности, этот подход позволяет обеспечить более высокую производительность системы и улучшить пользовательский опыт.


## Асинхронная обработка событий регистрации пользователей с использованием Kafka и статусов сообщений

Сервис использует Apache Kafka в качестве шины данных для асинхронной обработки событий, связанных с регистрацией пользователей. При успешной регистрации, сервис генерирует и отправляет сообщение в соответствующий Kafka-топик.

Процесс отправки сообщений:

1. Генерация сообщения: При регистрации пользователя сервис создает сообщение, содержащее релевантную информацию о пользователе.
2. Асинхронная отправка: Сообщение отправляется в Kafka-топик асинхронно, т.е. сервис не ждет подтверждения успешной доставки сообщения.
3. Сохранение статуса сообщения: При отправке сообщения в Kafka, сервис записывает в базу данных статус "inProgress", сигнализируя о том, что отправка была инициирована.
4. Обработка ответов от Kafka: Отдельная горутина (coroutine) обрабатывает канал ответов от Kafka, получая информацию об успешной или неудачной доставке сообщения.
5. Обновление статуса сообщения: В случае успешной доставки сообщения, горутина обновляет статус сообщения в базе данных на "Success".

Пример использования:

1. Сервис регистрации: При успешной регистрации пользователя, сервис публикует сообщение в Kafka-топик.
2. Сервис отправки сообщений: Подписан на топик. При получении сообщения, сервис обрабатывает данные нового пользователя и отправляет ему приветственное сообщение.
3. Сервис начисления баллов лояльности: Подписан на топик. При получении сообщения, сервис обрабатывает данные нового пользователя и начисляет ему приветственные баллы.

## Выбор библиотеки для взаимодействия сервисом через Kafka

Для взаимодействия между сервисами через Kafka используется формат данных protobuf. Преимуществом по сравнению с 
json является снижение кол-ва передаваемых байт (см. Высоконагруженные приложения. Программирование, масштабирование, поддержка. Автор
Мартин Клеппман). 

Было принято решение передачи данных через брокер Kafka с использованием   Schema Registry.  Schema Registry - это сервис, который позволяет хранить 
и управлять схемами данных, которые используются в Kafka. Это позволит:
* Управлять версиями схем: можно добавлять новые поля, изменять типы данных и т.д. без нарушения совместимости.
* Проверять данные: Schema Registry гарантирует, что данные, передаваемые в Kafka, соответствуют определенной схеме.
* Упростить десериализацию: Получатели данных могут использовать Schema Registry для получения необходимой схемы для десериализации данных.

Рассматривались две библиотеки в go. [Kafka-go](https://github.com/segmentio/kafka-go) и [Сonfluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go).
Kafka-go данный момент не имеет встроенной поддержки Schema Registry https://github.com/segmentio/kafka-go/issues/728#issuecomment-909690992 и https://github.com/segmentio/kafka-go/issues/728#issuecomment-2221492034. 
Чтобы не  разработать собственный механизм взаимодействия с Schema Registry принятно решение использовать Сonfluent-kafka-go.

Кратко:

* Schema Registry - это сервис, который позволяет хранить и управлять схемами данных, которые вы используете в Kafka. Это позволяет вам:
    * Управлять версиями схем: Вы можете добавлять новые поля, изменять типы данных и т.д. без нарушения совместимости.
    * Проверять данные: Schema Registry гарантирует, что данные, передаваемые в Kafka, соответствуют определенной схеме.
    * Упростить десериализацию: Получатели данных могут использовать Schema Registry для получения необходимой схемы для десериализации данных.

* Kafka-go - это библиотека Golang для работы с Kafka.

* Проблема: Библиотека Kafka-go не имеет встроенной поддержки Schema Registry. Это означает, что вам придется реализовать взаимодействие с Schema Registry вручную.

В результате:

* Вам нужно будет найти альтернативные библиотеки Golang, которые поддерживают Schema Registry, такие как `confluent-kafka-go` или `go-avro` .
* Либо разработать собственный механизм взаимодействия с Schema Registry.

Рекомендации:

1. Используйте библиотеки, поддерживающие Schema Registry:
    * `confluent-kafka-go` - официальная библиотека от Confluent, которая поддерживает Schema Registry (https://github.com/confluentinc/confluent-kafka-go).
    * `go-avro` - библиотека для работы с Avro, которая также поддерживает Schema Registry (https://github.com/linkedin/goavro).
2. Рассмотрите вариант ручного взаимодействия с Schema Registry:
    * Вы можете использовать HTTP-запросы для получения схемы из Schema Registry и затем использовать ее для сериализации/десериализации данных.
    * Но это потребует дополнительных усилий по реализации логики работы со Schema Registry.


################################################################################################## 

cd authloyalty/protos/proto/registration
protoc --go_out=. registration.proto

easyjson -all /home/alex/Dev/GolandYandex/authloyalty/internal/handlersapi/v1/sso_handlers_response.go


```bash
curl --header "Content-Type: application/json" --request POST --data '{"email":"test@test.com","password":"test"}' http://localhost:8000/auth/login
```

```bash
curl --header "Content-Type: application/json" --request POST --data '{"email":"test@test.com","password":"test"}' http://localhost:8000/auth/registration
```

```bash
curl --header "Content-Type: application/json" --request POST --data '{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InRlc3RAdGVzdC5jb20iLCJleHAiOjE3MjI0MzIwODMsInRva2VuX3R5cGUiOiJhY2Nlc3MiLCJ1aWQiOjJ9.J6XilG2yEAM611yybY8LdvXs046yrx8bjCoWlwd5dtQ"}' http://localhost:8000/auth/logout
```

// HOW TO ADD GRPC SWAGGER
https://apidog.com/articles/how-to-add-swagger-ui-for-grpc/

1. download buf bin from github
2. rename to buf
3. move to /usr/bin
4. chmod +x buf
5. buf generate 

if Failure: plugin openapiv2: could not find protoc plugin for name openapiv2 - please make sure protoc-gen-openapiv2 is installed and present on your $PATH
```bash
go install \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
    google.golang.org/protobuf/cmd/protoc-gen-go \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

# redis sentinel
https://redis.uptrace.dev/guide/go-redis-sentinel.html#redis-server-client

# auth swagger
http://localhost:8000/swagger/index.html

```
swag init -g ./cmd/sso/main.go -o ./cmd/sso/docs
```

if err when starts 

Golang swaggo rendering error: "Failed to load API definition" and "Fetch error doc.json" [closed]
Where the routers locate
n most cases, the problem is that you forgot to import the generated docs as _ "<your-project-package>/docs" 
in my case
_ "github.com/AlexBlackNn/authloyalty/cmd/sso/docs"


metrics
https://stackoverflow.com/a/65609042
https://github.com/prometheus/client_golang
https://grafana.com/oss/prometheus/exporters/go-exporter/#metrics-usage

// otel instrumentation
https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation#new-instrumentation
https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/github.com/gin-gonic/gin/otelgin/example/server.go
https://github.com/confluentinc/confluent-kafka-go/issues/712
https://github.com/etf1/opentelemetry-go-contrib

// kafka health check
https://github.com/confluentinc/confluent-kafka-go/discussions/1041

# using confluent-kafka-go
https://stackoverflow.com/a/55106860
CGO_ENABLED=0 can't be used, so dynamic links
https://github.com/confluentinc/confluent-kafka-go/issues/303?ysclid=lzrc7rstfd681525235#issuecomment-530566274
https://stackoverflow.com/a/69030479
https://github.com/confluentinc/confluent-kafka-go/issues/461

https://stackoverflow.com/questions/37630274/what-do-these-go-build-flags-mean-netgo-extldflags-lm-lstdc-static
https://blog.hashbangbash.com/2014/04/linking-golang-statically/ 



Решается флагами при сборке. Получившийся образ стал 68 МБ, что радует против 890МБ изначального 🤯

RUN GOOS=linux go build -ldflags '-extldflags "-static"' -o main ./cmd/sso/main.go

Do not link against shared libraries. This is only meaningful on platforms for which shared libraries are supported. The different variants of this option are for compatibility with various systems. You may use this option multiple times on the command line: it affects library searching for -l options which follow it. This option also implies --unresolved-symbols=report-all. This option can be used with -shared. Doing so means that a shared library is being created but that all of the library's external references must be resolved by pulling in entries from static libraries.
https://github.com/ko-build/ko/issues/756#issue-1298084220

githubassets.com
Build with -extldflags="-static" by default · Issue #756 · ko-build/ko · GitHub

-extldflags flags Set space-separated flags to pass to the external linker. from https://pkg.go.dev/cmd/link -static Do not link against shared libraries. This is only meaningful on platforms for which shared libraries are supported. The...

Вот еще крутой туториал нашел https://blog.hashbangbash.com/2014/04/linking-golang-statically/ Из интересного  ldd можно использовать чтоюы посмотреть динамическая или нет линковка. И собственно правильные флаги " go build --ldflags '-extldflags "-static"' ./code-cgo.go"

https://stackoverflow.com/a/76177689 -  а в этой ссылке у меня не сработало добавка -tags musl

GOOS=linux go build -tags musl -o main ./cmd/sso/main.go

/snap/go/10660/pkg/tool/linux_amd64/link: running gcc failed: exit status 1
/usr/bin/ld: /home/alex/go/pkg/mod/github.com/confluentinc/confluent-kafka-go@v1.9.2/kafka/librdkafka_vendor/librdkafka_musl_linux.a(rdkafka_admin.o): in function `rd_kafka_CreateTopicsResponse_parse':
(.text+0x730): undefined reference to `strlcpy'
/usr/bin/ld: (.text+0x834): undefined reference to `strlcpy'
/usr/bin/ld: (.text+0xcef): undefined reference to `strlcpy'
/usr/bin/ld: (.text+0xe7d): undefined reference to `strlcpy'
/usr/bin/ld: (.text+0x1014): undefined reference to `strlcpy'