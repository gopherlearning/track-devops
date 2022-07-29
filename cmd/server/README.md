# cmd/agent

В данной директории будет содержаться код Сервера, который скомпилируется в бинарное приложение




## docker
```bash
sudo docker run --name track-devops-postgres -e POSTGRES_PASSWORD=mysecretpassword -p 127.0.0.1:13131:5432 -d postgres

sudo docker exec -it -u postgres track-devops-postgres psql


go install github.com/swaggo/swag/cmd/swag@latest


CREATE TABLE metrics (  
  target VARCHAR ( 50 ) UNIQUE NOT NULL,
  data jsonb NOT NULL
);

DROP TABLE metrics;


select * from metrics ;


sudo docker kill track-devops-postgres
sudo docker rm track-devops-postgres

# run with storage
go run cmd/server/main.go -a=127.0.0.1:1212 -f=/tmp/bla -i=5s -r=true -k=bhygyg


# run with db
go run cmd/server/main.go -a=127.0.0.1:1212 -f=/tmp/bla -i=5s -d=postgres://postgres:mysecretpassword@localhost:13131/postgres?sslmode=disable -r=true -k=bhygyg

# for profiling
for i in {1..50000}; do curl http://127.0.0.1:1212/ -o /dev/null --silent; done

```