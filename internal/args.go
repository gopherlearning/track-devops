package internal

import (
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
)

type ServerArgs struct {
	Verbose            bool          `name:"verbose" short:"v" help:"Включить расширенное логирование" env:"VERBOSE"`
	Config             string        `name:"config" short:"c" help:"Путь к файлу конфигурации" env:"CONFIG"`
	ServerAddr         string        `mapstructure:"address,omitempty" short:"a" help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	StoreFile          string        `mapstructure:"store_file,omitempty" short:"f" help:"строка, имя файла, где хранятся значения (пустое значение — отключает функцию записи на диск)" env:"STORE_FILE"`
	DatabaseDSN        string        `mapstructure:"database_dsn,omitempty" short:"d" help:"строка c адресом подключения к БД" env:"DATABASE_DSN"`
	Key                string        `mapstructure:"key,omitempty" short:"k" help:"Ключ подписи" env:"KEY"`
	Restore            bool          `mapstructure:"restore,omitempty" short:"r" help:"булево значение (true/false), определяющее, загружать или нет начальные значения из указанного файла при старте сервера" env:"RESTORE"  default:"true"`
	UsePprof           bool          `mapstructure:"-" help:"Использовать Pprof" env:"PPROF"`
	ShowStore          bool          `mapstructure:"-" help:"Переодически выводить содержимое в консоль"`
	StoreInterval      time.Duration `mapstructure:"store_interval,omitempty" short:"i" help:"интервал времени в секундах, по истечении которого текущие показания сервера сбрасываются на диск (значение 0 — делает запись синхронной)" env:"STORE_INTERVAL"  default:"400s"`
	GenerateCryptoKeys bool          `mapstructure:"-" help:"Сгенерировать ключи для ассиметричного шифрования"`
	CryptoKey          string        `mapstructure:"crypto_key,omitempty" help:"Путь к файлу, где хранятся приватный ключ шифрования" env:"CRYPTO_KEY"`
}

type AgentArgs struct {
	Verbose        bool          `mapstructure:"verbose,omitempty" short:"v" help:"Включить расширенное логирование" env:"VERBOSE"`
	Config         string        `mapstructure:"-" short:"c" help:"Путь к файлу конфигурации" name:"config" env:"CONFIG"`
	ServerAddr     string        `mapstructure:"address,omitempty" short:"a" help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	Key            string        `mapstructure:"key,omitempty" short:"k" help:"Ключ подписи" env:"KEY"`
	Format         string        `mapstructure:"format,omitempty" short:"f" help:"Report format" env:"FORMAT"`
	Batch          bool          `mapstructure:"batch,omitempty" short:"b" help:"Send batch mrtrics" env:"BATCH" default:"true"`
	PollInterval   time.Duration `mapstructure:"poll_interval,omitempty" short:"p" help:"Poll interval" env:"POLL_INTERVAL" default:"2s"`
	ReportInterval time.Duration `mapstructure:"report_interval,omitempty" short:"r" help:"Report interval" env:"REPORT_INTERVAL" default:"10s"`
	CryptoKey      string        `mapstructure:"crypto_key,omitempty" help:"Путь к файлу, где хранятся публийчный ключ шифрования" env:"CRYPTO_KEY" default:""`
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
}

// FixArgs исправляет аргументы командной строки и возвращает путь к конфигу, если он занят
func FixArgs() string {
	var confPath string
	// только для прохождения теста
	for i := 0; i < len(os.Args); i++ {
		if strings.Contains(os.Args[i], "=") {
			a := strings.Split(os.Args[i], "=")
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
