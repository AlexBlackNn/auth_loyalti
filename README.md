# otelconfluent - это модифицированный код https://github.com/etf1/opentelemetry-go-contrib
# возможно сделаю форк и подправлю его в другой репе. Подробный комментарий в пачке. 

# SSO  - Сервис авторизации

Единый сервис авторизации (СА) и управления пользователями 

## Запуск

### Локальный  
```bash
cd commands && task local
```

### На Демо стенде
```bash
cd commands && task demo
```

### Тестов интеграционных
 
```bash
cd commands && task integration-tests 
```

### Юнит Тестов
```bash
cd commands && task unit-tests 
```

## Swagger grpc доступен по адресу:
http://localhost:44044/sw/

## Swagger http доступен по адресу:
http://localhost:8000/swagger/index.html

## Grafana
http://localhost:3000/grafana/
Для просмотра метрик, в папке monitoring лежит dashboard  - 6671_rev2.json

## Prometheus
http://localhost:9090/targets/

## Kafka UI
http://localhost:8080

##  metrics
http://localhost:8000/metrics 


----------------------------------------------------
БУДЕТ ПЕРЕПИСАНО. В РАБОТЕ.
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


## Секционирование БД 
В распределенной базе данных, состоящей из нескольких узлов, нагрузка со временем может меняться. Причины:

* Увеличение количества запросов: Требуются дополнительные вычислительные ресурсы для обработки.
* Рост объема данных: Необходимы дополнительные хранилища (жесткие диски, оперативная память).
* Сбои узлов: Другие узлы берут на себя нагрузку выбывшего узла.

У меня сделано партиционирование таблицы с пользователями. Партиции сделал по email (у меня основные запросы завязаны на email One of the most critical design decisions will be the column or columns by which you partition your data. Often the best choice will be to partition by the column or set of columns which most commonly appear in WHERE clauses of queries being executed on the partitioned table https://www.postgresql.org/docs/current/ddl-partitioning.html#DDL-PARTITIONING-DECLARATIVE-BEST-PRACTICES).  Пишу всегда в мастер, читаю из слейвов. Партиции, за счет репликации находятся на каждом из слейвов.  Потребуется ли какая-то перебалансировка в будущем? Думаю возможно получиться что часть партиций будет более горячими.

Перебалансировка должна:
1. Обеспечивать равномерное распределение нагрузки: После перебалансировки нагрузка (хранение данных, чтение/запись) должна быть равномерно распределена по всем узлам. (это автоматом будет за счет репликации и чтнеия из реплик,  разве что равномерная должна быть нагрузка на партиции, чтобы быстрее можно было искать.)
2. Обеспечивать доступность данных: База данных должна оставаться доступной для операций чтения и записи во время перебалансировки.
3. Минимизировать перемещение данных: Перемещается только необходимое количество данных, чтобы ускорить процесс и уменьшить нагрузку на сеть и хранилища.

Если бы партиции хранились бы частично на одной машине частично на другой, то можно было бы  создать намного больше партиций, чем узлов в системе, и распределить по нескольку партиций на каждый узел.  
Кластер из 3х узлов (мастер и два слейва) можно разделить было бы  на 1000 партиций. Тогда добавляемый в кластер новый узел может взять по нескольку партиций у каждого из существующих узлов на время, до тех пор пока партиции не станут снова распределены равномерно. Между узлами перемещались бы только партиции целиком. Их количество  не менялось бы, как и соответствие ключей партициям. Единственное, что менялось бы -
распределение их по узлам.  Но за счет репликации смысла в этом нет.

Тогда надо собрать статистику по нагрзуке на партицию и сделать перепартиционирование на мастере. И реплики подхватят эти именения. Можно ли это сделать не блакируя работу пользователей? Например сделать перебалансировку на реплике (выключив ее из кластера) и поднять ее до мастера. Но тогда потеряем часть данных, которые были записаны в мастер во время того, как мы делали репартиционирование.
5. ################################################################################################## 

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


docker compose -f docker-compose.prod.yaml build --progress=plain --no-cache
docker compose -f docker-compose.prod.yaml up


metrics - 4 golden signals
https://github.com/slok/go-http-metrics?tab=readme-ov-file#benchmarks 

### kafka tracing transfer 

https://stackoverflow.com/a/78329944
https://opentelemetry.io/docs/demo/architecture/
https://github.com/open-telemetry/opentelemetry-demo/tree/e5c45b9055627795e7577c395c641f6cf240f054
https://github.com/open-telemetry/opentelemetry-demo/blob/e5c45b9055627795e7577c395c641f6cf240f054/src/checkoutservice/main.go#L527
https://www.youtube.com/watch?v=49fA7gQsDwA&t=2539s
https://www.youtube.com/watch?v=5rjTdA6BM1E
https://www.youtube.com/watch?v=UEwkn0iHDzA&list=PLNxnp_rzlqf6z1cC0IkIwp6yjsBboX945&index=1
## SSO

### NEED TO INSTALL
1. Protocol Buffer Compiler Installation
   https://grpc.io/docs/protoc-installation/#install-using-a-package-manager Г

$ apt install -y protobuf-compiler
$ protoc --version  # Ensure compiler version is 3+

2. Install go plugins
   https://grpc.io/docs/languages/go/quickstart/

3. Генерим код:
   ~/GolandProjects/sso/protos$ protoc -I proto proto/sso/sso.proto --go_out=./gen/go --go_opt=paths=source_relative --go-grpc_out=./gen/go/ --go-grpc_opt=paths=source_relative
>>> ./gen/go/: No such file or directory

~/GolandProjects/sso/protos$
protoc -I proto proto/sso/sso.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

Сгенирированные файлы будут лежать в protos/gen/go/sso

4. Для автоматизации установить task
   https://taskfile.dev/api/
   sudo snap install task --classic

   далее просто делаем >>>  task generate
   alex@black:~/GolandProjects/sso/protos$ task generate
   task: [generate] protoc -I proto proto/sso/*.proto --go_out=./gen/go/ --go_opt=paths=source_relative --go-grpc_out=./gen/go/ --go-grpc_opt=paths=source_relative


auth swagger

http://localhost:8000/swagger/index.html

swag init -g ./cmd/sso/main.go -o ./cmd/sso/docs

if err when starts

Golang swaggo rendering error: "Failed to load API definition" and "Fetch error doc.json" [closed] Where the routers locate n most cases, the problem is that you forgot to import the generated docs as _ "/docs" in my case _ "github.com/AlexBlackNn/authloyalty/cmd/sso/docs"

metrics https://stackoverflow.com/a/65609042 https://github.com/prometheus/client_golang https://grafana.com/oss/prometheus/exporters/go-exporter/#metrics-usage