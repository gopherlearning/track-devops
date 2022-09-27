# cmd/agent

В данной директории будет содержаться код Агента, который скомпилируется в бинарное приложение



```bash

# run with storage
go run cmd/agent/main.go -a=127.0.0.1:1212 -r=3s -k=bhygyg -f=json

# run with crypto key
go run cmd/agent/main.go -a="127.0.0.1:1212" -r=3s -k=bhygyg -f=json --crypto-key="key.pub"

# run with config
go run cmd/agent/main.go -c="cmd/agent/config.json"
```