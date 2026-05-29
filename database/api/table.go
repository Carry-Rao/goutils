package api

type Table interface {
	Create(map[string]any) error
	Get(map[string]any) ([]any, error)
	Set(map[string]any) error
	Delete(map[string]any) error
}
