package internal

import (
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v6"
)

type ServerArgs struct {
	Verbose            bool          `name:"verbose" short:"v" help:"Включить расширенное логирование" env:"VERBOSE"`
	Config             string        `name:"config" json:"-" short:"c" help:"Путь к файлу конфигурации" env:"CONFIG"`
	ServerAddr         string        `name:"address" short:"a" help:"Server address" env:"ADDRESS" default:"127.0.0.1:8080"`
	StoreFile          string        `name:"store-file" json:"store_file" short:"f" help:"строка, имя файла, где хранятся значения (пустое значение — отключает функцию записи на диск)" env:"STORE_FILE" default:"/tmp/devops-metrics-db.json"`
	DatabaseDSN        string        `name:"database-dsn" json:"database_dsn" short:"d" help:"строка c адресом подключения к БД" env:"DATABASE_DSN"`
	Key                string        `name:"key" short:"k" help:"Ключ подписи" env:"KEY"`
	Restore            bool          `name:"restore" short:"r" help:"булево значение (true/false), определяющее, загружать или нет начальные значения из указанного файла при старте сервера" env:"RESTORE"  default:"true"`
	UsePprof           bool          `help:"Использовать Pprof" env:"PPROF"`
	ShowStore          bool          `help:"Переодически выводить содержимое в консоль"`
	StoreInterval      time.Duration `name:"store-interval" json:"store_interval" short:"i" help:"интервал времени в секундах, по истечении которого текущие показания сервера сбрасываются на диск (значение 0 — делает запись синхронной)" env:"STORE_INTERVAL"  default:"400s"`
	GenerateCryptoKeys bool          `help:"Сгенерировать ключи для ассиметричного шифрования"`
	CryptoKey          string        `name:"crypto-key" json:"crypto_key" help:"Путь к файлу, где хранятся приватный ключ шифрования" env:"CRYPTO_KEY"`
	TrustedSubnet      string        `name:"trusted-subnet" json:"trusted_subnet" short:"t" help:"Доверенные сети" env:"TRUSTED_SUBNET"`
}

type AgentArgs struct {
	Verbose        bool          `name:"verbose" short:"v" help:"Включить расширенное логирование" env:"VERBOSE"`
	Config         string        `name:"config" json:"-" short:"c" help:"Путь к файлу конфигурации" env:"CONFIG"`
	ServerAddr     string        `name:"address" short:"a" help:"Server address" env:"ADDRESS" default:"127.0.0.1:8080"`
	Key            string        `name:"key" short:"k" help:"Ключ подписи" env:"KEY"`
	Format         string        `name:"format" short:"f" help:"Report format" env:"FORMAT"`
	Batch          bool          `name:"batch" short:"b" help:"Send batch mrtrics" env:"BATCH" default:"true"`
	PollInterval   time.Duration `name:"poll-interval" json:"poll_interval" short:"p" help:"Poll interval" env:"POLL_INTERVAL" default:"2s"`
	ReportInterval time.Duration `name:"report-interval" json:"report_interval" short:"r" help:"Report interval" env:"REPORT_INTERVAL" default:"10s"`
	CryptoKey      string        `name:"crypto-key" json:"crypto_key" help:"Путь к файлу, где хранятся публийчный ключ шифрования" env:"CRYPTO_KEY"`
	SelfAddress    string        `name:"self-address" json:"self_address" help:"Адрес, используемы в качестве исходящего, для отправки запросов к серверу" env:"CRYPTO_KEY" default:"127.0.0.1"`
}

// ReadConfig задаёт стандартные значения, читает конфиг, проверяет переменное окружение и флаги
func ReadConfig(cfg interface{}) {
	opts := []kong.Option{
		kong.Name("server"),
		kong.Description("desc"),
		kong.UsageOnError(),
	}
	if path := FixArgs(); len(path) != 0 {
		opts = append(opts, kong.Configuration(kong.JSON, path))
	}
	parser := kong.Must(cfg, opts...)
	_, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)
	err = env.Parse(cfg)
	parser.FatalIfErrorf(err)
}

// FixArgs исправляет аргументы командной строки и возвращает путь к конфигу, если он занят
func FixArgs() string {
	var confPath string
	// только для прохождения теста
	for i := 0; i < len(os.Args); i++ {
		if strings.Contains(os.Args[i], "=") {
			a := strings.Split(os.Args[i], "=")
			if !strings.Contains(a[0], "-") {
				continue
			}
			if len(a) > 2 {
				b := strings.Join(a[1:], "=")
				a = []string{a[0], b}
			}
			if a[0] == "-r" && (a[1] == "false" || a[1] == "true") {
				os.Args[i] = strings.ReplaceAll(os.Args[i], "-r", "--restore")
				continue
			}
			os.Args = append(os.Args[:i], append(a, os.Args[i+1:]...)...)
		}
	}
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i][:1] == "-" && os.Args[i][1:2] != "-" && len(os.Args[i][1:]) != 1 {
			os.Args[i] = strings.Replace(os.Args[i], "-", "--", 1)
		}
		if os.Args[i] == "-c" || os.Args[i] == "--config" {
			confPath = os.Args[i+1]
		}
	}
	return confPath
}
