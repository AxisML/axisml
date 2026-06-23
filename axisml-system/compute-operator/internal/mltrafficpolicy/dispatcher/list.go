package dispatcher

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// newListFor returns a freshly-allocated *List object for the given prototype,
// resolving the GVK via the scheme. Mirrors the MLService dispatcher helper.
func newListFor(proto client.Object, scheme *runtime.Scheme) (client.ObjectList, error) {
	gvks, _, err := scheme.ObjectKinds(proto)
	if err != nil {
		return nil, fmt.Errorf("look up GVK for %T: %w", proto, err)
	}
	if len(gvks) == 0 {
		return nil, fmt.Errorf("no GVK registered for %T", proto)
	}
	gvk := gvks[0]
	gvk.Kind += "List"
	obj, err := scheme.New(gvk)
	if err != nil {
		return nil, fmt.Errorf("instantiate %s: %w", gvk, err)
	}
	list, ok := obj.(client.ObjectList)
	if !ok {
		return nil, fmt.Errorf("instantiated %T is not a client.ObjectList", obj)
	}
	return list, nil
}

// extractItems unrolls a List object into individual client.Objects.
func extractItems(list client.ObjectList) ([]client.Object, error) {
	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, fmt.Errorf("extract list items: %w", err)
	}
	out := make([]client.Object, 0, len(items))
	for _, it := range items {
		obj, ok := it.(client.Object)
		if !ok {
			v := reflect.ValueOf(it)
			if v.Kind() != reflect.Pointer {
				ptr := reflect.New(v.Type())
				ptr.Elem().Set(v)
				if obj, ok = ptr.Interface().(client.Object); !ok {
					return nil, fmt.Errorf("list item %T is not a client.Object", it)
				}
			} else {
				return nil, fmt.Errorf("list item %T is not a client.Object", it)
			}
		}
		out = append(out, obj)
	}
	return out, nil
}
