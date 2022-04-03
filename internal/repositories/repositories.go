package repositories

type Repository interface {
	Get(target, metric, name string) string
	Update(target, metric, name, value string) error
	List(targets ...string) map[string][]string
	ListProm(targets ...string) []byte
}
