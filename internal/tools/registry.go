package tools

// Tool represents a single dev-tool exposed to the frontend (e.g. JSON
// formatter, XML diff). Run receives the primary input plus free-form
// options and returns the transformed output or an error.
type Tool struct {
	ID   string
	Name string
	Run  func(input string, opts map[string]string) (string, error)
}

var registry = map[string]*Tool{}

// Register adds a tool to the registry. Tool packages call this from an
// init() function so the shell never needs to know about them directly.
func Register(t *Tool) {
	registry[t.ID] = t
}

// Get returns the tool for the given ID, or nil if it isn't registered.
func Get(id string) *Tool {
	return registry[id]
}

// List returns all registered tools, in no particular order.
func List() []*Tool {
	list := make([]*Tool, 0, len(registry))
	for _, t := range registry {
		list = append(list, t)
	}
	return list
}

// DiffTool represents a two-input comparator (e.g. JSON diff, XML diff).
type DiffTool struct {
	ID   string
	Name string
	Run  func(left, right string) (string, error)
}

var diffRegistry = map[string]*DiffTool{}

// RegisterDiff adds a diff tool to the registry.
func RegisterDiff(t *DiffTool) {
	diffRegistry[t.ID] = t
}

// GetDiff returns the diff tool for the given ID, or nil if it isn't registered.
func GetDiff(id string) *DiffTool {
	return diffRegistry[id]
}

// ListDiff returns all registered diff tools, in no particular order.
func ListDiff() []*DiffTool {
	list := make([]*DiffTool, 0, len(diffRegistry))
	for _, t := range diffRegistry {
		list = append(list, t)
	}
	return list
}
