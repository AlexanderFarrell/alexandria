package apis

// Server is the interface any HTTP (or other transport) implementation must satisfy.
type Server interface {
	Start(addr string) error
	Shutdown() error
}
