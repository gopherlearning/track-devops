package repositories

import "errors"

var (
	ErrWrongMetricURL      = errors.New("неправильный формат метрики")
	ErrWrongMetricType     = errors.New("нет метрики такого типа")
	ErrWrongMetricValue    = errors.New("неверное значение метрики")
	ErrWrongTarget         = errors.New("неправильный источник метрик")
	ErrWrongValueInStorage = errors.New("ошибка в хранилище")
)
