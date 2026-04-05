package cas

// TagStore defines the interface for the tag system.
// Tags associate human-readable name@version pairs with content references.
type TagStore interface {
	// Tag assigns a name@version to a content reference.
	Tag(name string, version string, ref Ref) error

	// Resolve looks up a tag to its content reference.
	Resolve(name string, version string) (Ref, error)

	// ListTags returns all tags matching a prefix.
	ListTags(prefix string) ([]TagEntry, error)
}
