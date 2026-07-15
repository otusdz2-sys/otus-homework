# user-service (ДЗ №4 «Инфраструктурные паттерны»)

RESTful CRUD-сервис пользователей на Go с PostgreSQL. Конфигурация приложения хранится
в ConfigMap, доступы к БД — в Secret, первоначальная миграция оформлена Job-ой,
ингресс ведёт на `arch.homework`, как и в прошлом задании.

Образ собран под `linux/amd64`, опубликован на Docker Hub как `cornsc/user-service:1.0`
и в Deployment указан именно оттуда.

## API

- `POST /user` — создать пользователя, тело: `username`, `firstName`, `lastName`, `email`, `phone`
- `GET /user/{id}` — получить пользователя
- `PUT /user/{id}` — обновить пользователя
- `DELETE /user/{id}` — удалить пользователя
- `GET /health` — liveness-проба
- `GET /ready` — readiness-проба, проверяет доступность БД

## Состав

```
hw04-helm/
├── app/                         # исходники сервиса и Dockerfile
├── k8s/                         # манифесты, нумерация задаёт порядок применения
│   ├── 01-configmap.yaml        # конфигурация приложения
│   ├── 02-secret.yaml           # доступы к БД
│   ├── 03-migration-job.yaml    # первоначальная миграция (Job)
│   ├── 04-deployment.yaml       # 2 реплики + liveness/readiness пробы
│   ├── 05-service.yaml          # ClusterIP :8000
│   └── 06-ingress.yaml          # arch.homework/user
├── helm/
│   ├── postgres-values.yaml     # values для чарта bitnami/postgresql
│   └── user-service/            # задание со звездой: helm-чарт приложения
└── postman/user-service.postman_collection.json
```

## Сборка образа

```bash
cd app
docker build --platform linux/amd64 -t cornsc/user-service:1.0 .
docker push cornsc/user-service:1.0
```

## Установка БД из helm

Старый репозиторий `charts.bitnami.com` закрыт (отдаёт 403), чарты bitnami переехали
в OCI-реестр Docker Hub, поэтому ставим оттуда:

```bash
helm install postgres oci://registry-1.docker.io/bitnamicharts/postgresql -f helm/postgres-values.yaml
```

В `helm/postgres-values.yaml` заданы пользователь, пароль, имя базы и ресурсы.
Сервис БД получает имя `postgres-postgresql`, оно же прописано в ConfigMap приложения.

## Развёртывание приложения

```bash
kubectl apply -f k8s/
```

`kubectl` применяет файлы в алфавитном порядке, так что нумерация в именах задаёт
правильную последовательность: конфиг и секрет, затем миграция, затем деплоймент
с сервисом и ингрессом.

Команда применения первоначальной миграции отдельно:

```bash
kubectl apply -f k8s/03-migration-job.yaml
kubectl wait --for=condition=complete job/user-service-migration
```

Миграция идемпотентна (`CREATE TABLE IF NOT EXISTS`), а `backoffLimit: 10` позволяет
Job-е спокойно пережить запуск раньше готовности БД. Для повторного запуска сначала
удалить старую Job-у: `kubectl delete job user-service-migration`.

## Задание со звездой: helm-чарт приложения

Те же манифесты шаблонизированы в чарте `helm/user-service`: образ, число реплик,
хост ингресса, конфигурация и доступы к БД вынесены в values.yaml. Вместо
`kubectl apply -f k8s/`:

```bash
helm install user-service helm/user-service
```

Миграция в чарте оформлена post-install/post-upgrade hook-ом и выполняется сама
при установке и каждом обновлении релиза.

## Удаление

```bash
kubectl delete -f k8s/            # либо helm uninstall user-service, если ставили чартом
helm uninstall postgres
kubectl delete pvc data-postgres-postgresql-0   # том с данными БД
```

## Доступ по имени arch.homework

Как в прошлом ДЗ. Nginx ingress controller ставится через helm:

```bash
kubectl create namespace m
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx/
helm install nginx ingress-nginx/ingress-nginx --namespace m
```

На macOS с driver=docker дополнительно нужны строка `127.0.0.1 arch.homework`
в `/etc/hosts` и запущенный `minikube tunnel`.

## Проверка вручную

```bash
curl -X POST http://arch.homework/user -H 'Content-Type: application/json' \
  -d '{"username":"johndoe","firstName":"John","lastName":"Doe","email":"john@example.com","phone":"+79990001122"}'
curl http://arch.homework/user/1
curl -X PUT http://arch.homework/user/1 -H 'Content-Type: application/json' \
  -d '{"username":"johndoe","firstName":"Johnny","lastName":"Doe","email":"johnny@example.com","phone":"+79990001122"}'
curl -X DELETE http://arch.homework/user/1
```

## Postman / newman

Базовый url в коллекции — `arch.homework`. Запуск:

```bash
newman run postman/user-service.postman_collection.json
```

> Если на машине задан системный `HTTP_PROXY`/`HTTPS_PROXY`, запускать в обход прокси:
> `env -u HTTP_PROXY -u HTTPS_PROXY -u http_proxy -u https_proxy newman run ...`

Вывод прогона:

```
OTUS user-service CRUD

→ Create user
  POST http://arch.homework/user [200 OK, 249B, 30ms]
  ✓  status 200
  ✓  user has id
  ✓  username saved

→ Get user
  GET http://arch.homework/user/4 [200 OK, 249B, 4ms]
  ✓  status 200
  ✓  username matches
  ✓  firstName matches

→ Update user
  PUT http://arch.homework/user/4 [200 OK, 253B, 4ms]
  ✓  status 200
  ✓  firstName updated

→ Get user after update
  GET http://arch.homework/user/4 [200 OK, 253B, 6ms]
  ✓  status 200
  ✓  firstName is updated in db
  ✓  email is updated in db

→ Delete user
  DELETE http://arch.homework/user/4 [204 No Content, 88B, 3ms]
  ✓  status 204

→ Get deleted user
  GET http://arch.homework/user/4 [404 Not Found, 166B, 7ms]
  ✓  status 404
  ✓  error message

┌─────────────────────────┬─────────────────┬─────────────────┐
│                         │        executed │          failed │
├─────────────────────────┼─────────────────┼─────────────────┤
│              iterations │               1 │               0 │
├─────────────────────────┼─────────────────┼─────────────────┤
│                requests │               6 │               0 │
├─────────────────────────┼─────────────────┼─────────────────┤
│            test-scripts │               6 │               0 │
├─────────────────────────┼─────────────────┼─────────────────┤
│      prerequest-scripts │               0 │               0 │
├─────────────────────────┼─────────────────┼─────────────────┤
│              assertions │              14 │               0 │
├─────────────────────────┴─────────────────┴─────────────────┤
│ total run duration: 128ms                                   │
├─────────────────────────────────────────────────────────────┤
│ total data received: 499B (approx)                          │
├─────────────────────────────────────────────────────────────┤
│ average response time: 9ms [min: 3ms, max: 30ms, s.d.: 9ms] │
└─────────────────────────────────────────────────────────────┘
```
