// Cache is an interface for cache implementation.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}