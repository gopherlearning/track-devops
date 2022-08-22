# go-musthave-devops-tpl

Шаблон репозитория для практического трека «Go в DevOps».

# Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` - адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

# Обновление шаблона

Чтобы получать обновления автотестов и других частей шаблона, выполните следующую команду:

```
git remote add -m main template https://github.com/yandex-praktikum/go-musthave-devops-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/main .github
```

Затем добавьте полученные изменения в свой репозиторий.

### Проверка покрытия тестами
```bash
go test ./... -coverprofile=profile.cov
go tool cover -func profile.cov


go tool pprof -http=":9090" -seconds=30 http://127.0.0.1:1212/debug/pprof/profile




curl -sK -v http://localhost:1212/debug/pprof/profile > profiles/base.pprof

curl -sK -v http://localhost:1212/debug/pprof/profile > profiles/result.pprof

```

### Запуск multichecker
```bash

go run cmd/staticlint/main.go ./..

```