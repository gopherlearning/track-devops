package internal

import "time"

type ServerArgs struct {
	Verbose            bool          ` short:"v" help:"Включить расширенное логирование" env:"VERBOSE"`
	ServerAddr         string        `short:"a" help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	StoreFile          string        `short:"f" help:"строка, имя файла, где хранятся значения (пустое значение — отключает функцию записи на диск)" env:"STORE_FILE" default:"/tmp/devops-metrics-db.json"`
	DatabaseDSN        string        `short:"d" help:"строка с адресом подключения к БД" env:"DATABASE_DSN"`
	Key                string        `short:"k" help:"Ключ подписи" env:"KEY"`
	Restore            bool          `short:"r" help:"булево значение (true/false), определяющее, загружать или нет начальные значения из указанного файла при старте сервера" env:"RESTORE" default:"true"`
	UsePprof           bool          `help:"Использовать Pprof" env:"PPROF"`
	ShowStore          bool          `help:"Переодически выводить содержимое в консоль"`
	StoreInterval      time.Duration `short:"i" help:"интервал времени в секундах, по истечении которого текущие показания сервера сбрасываются на диск (значение 0 — делает запись синхронной)" env:"STORE_INTERVAL" default:"300s"`
	GenerateCryptoKeys bool          `help:"Сгенерировать ключи для ассиметричного шифрования"`
	CryptoKey          string        `help:"Путь к файлу, где хранятся приватный ключ шифрования" env:"CRYPTO_KEY" default:"key.pem"`
}

type AgentArgs struct {
	Verbose        bool          ` short:"v" help:"Включить расширенное логирование" env:"VERBOSE"`
	ServerAddr     string        `short:"a" help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	Key            string        `short:"k" help:"Ключ подписи" env:"KEY"`
	Format         string        `short:"f" help:"Report format" env:"FORMAT"`
	Batch          bool          `short:"b" help:"Send batch mrtrics" env:"BATCH" default:"true"`
	PollInterval   time.Duration `short:"p" help:"Poll interval" env:"POLL_INTERVAL" default:"2s"`
	ReportInterval time.Duration `short:"r" help:"Report interval" env:"REPORT_INTERVAL" default:"10s"`
	CryptoKey      string        `help:"Путь к файлу, где хранятся публийчный ключ шифрования" env:"CRYPTO_KEY" default:"key.pub"`
}
