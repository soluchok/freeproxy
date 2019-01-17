package freeproxy

type Provider interface {
	List() ([]string, error)
	Name() string
}
