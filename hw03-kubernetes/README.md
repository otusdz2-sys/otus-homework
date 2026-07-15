# health-service (ДЗ №3 «Основы работы с Kubernetes»)

Минимальный сервис на Go: отвечает на порту `8000`, метод `GET /health` (и `GET /health/`)
возвращает `{"status": "OK"}`. Образ собран под `linux/amd64`, опубликован на Docker Hub
как `cornsc/health-service:1.1` и в Deployment указан именно оттуда.

## Состав

```
hw03-kubernetes/
├── main.go            # сервис
├── go.mod
├── Dockerfile         # сборка под linux/amd64
├── k8s/               # манифесты (применяются одной командой)
│   ├── deployment.yaml      # 2 реплики + liveness/readiness пробы
│   ├── service.yaml         # ClusterIP :8000
│   ├── ingress.yaml         # host arch.homework, путь /health
│   └── ingress-otusapp.yaml # задание со звездой: rewrite /otusapp/gustinov/* -> /*
└── postman/health-service.postman_collection.json
```

## Сборка и публикация образа

```bash
docker build --platform linux/amd64 -t cornsc/health-service:1.1 .
docker push cornsc/health-service:1.1
```

## Ingress-контроллер (nginx через helm)

```bash
kubectl create namespace m
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx/
helm repo update
helm install nginx ingress-nginx/ingress-nginx --namespace m
```

## Развёртывание приложения

```bash
kubectl apply -f k8s/
kubectl rollout status deployment/health-service
```

## Удаление

```bash
kubectl delete -f k8s/
# контроллер (если нужно убрать совсем):
helm uninstall nginx -n m && kubectl delete namespace m
```

## Доступ по имени arch.homework

**Linux** (routable minikube ip): в `/etc/hosts` добавить `<minikube ip> arch.homework`.

**macOS, driver=docker** (ip миникуба не маршрутизируется с хоста): нужен `minikube tunnel`.

```bash
echo "127.0.0.1 arch.homework" | sudo tee -a /etc/hosts
minikube tunnel        # оставить запущенным в отдельном терминале (спросит sudo)
```

## Проверка

```bash
curl http://arch.homework/health            # {"status": "OK"}
curl http://arch.homework/health/           # {"status": "OK"}
curl http://arch.homework/otusapp/gustinov/health   # {"status": "OK"}, задание со звездой
```

## Postman / newman

```bash
newman run postman/health-service.postman_collection.json
```

> Если на машине задан системный `HTTP_PROXY`/`HTTPS_PROXY`, локальный `arch.homework`
> уйдёт через прокси (можно поймать постороннюю 302). Тогда запускать в обход прокси:
> `env -u HTTP_PROXY -u HTTPS_PROXY -u http_proxy -u https_proxy newman run postman/health-service.postman_collection.json`
