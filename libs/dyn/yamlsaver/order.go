package yamlsaver

import "slices"

// This struct is used to generate indexes for ordering of map keys.
// The ordering defined based on any predefined Order in `Order` field
// or running Order based on `index`
type Order struct {
	index int
	order []string
}

func NewOrder(o []string) *Order {
	return &Order{index: 0, order: o}
}

// Returns an integer which represents the order of map key in resulting
// The lower the index, the earlier in the list the key is.
// If the order is not predefined, it uses running order and any subsequential call to
// order.Get returns an increasing index.
func (o *Order) Get(key string) int {
	index := slices.Index(o.order, key)
	// If the key is found in predefined order list
	// We return a negative index which put the value at the top of the order compared to other
	// not predefined keys. The earlier value in predefined list, the lower negative index value
	if index != -1 {
		return index - len(o.order)
	}

	// Otherwise we just increase the order index
	o.index += 1
	return o.index
}
