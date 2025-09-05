package informers

import (
	"k8s.io/client-go/tools/cache"
)

// genericInformer implements GenericInformer.
type genericInformer struct {
	informer cache.SharedIndexInformer
	lister   cache.GenericLister
}

// Informer returns the underlying SharedIndexInformer.
func (g *genericInformer) Informer() cache.SharedIndexInformer {
	return g.informer
}

// Lister returns a GenericLister for this resource.
func (g *genericInformer) Lister() cache.GenericLister {
	return g.lister
}
