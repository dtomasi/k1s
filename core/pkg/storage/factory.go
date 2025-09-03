package storage

// SimpleFactory implements the Factory interface by delegating to a provided Backend
type SimpleFactory struct {
	backend Backend
}

// NewSimpleFactory creates a factory that always returns the same backend
func NewSimpleFactory(backend Backend) Factory {
	return &SimpleFactory{
		backend: backend,
	}
}

// Create creates a new storage instance implementing the Interface
func (f *SimpleFactory) Create(_ Config) (Interface, error) {
	return f.backend, nil
}

// CreateBackend creates a new backend storage instance
func (f *SimpleFactory) CreateBackend(_ Config) (Backend, error) {
	return f.backend, nil
}

// SupportedBackends returns the list of supported backend types
func (f *SimpleFactory) SupportedBackends() []string {
	return []string{f.backend.Name()}
}
