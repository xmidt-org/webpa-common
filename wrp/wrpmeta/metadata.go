package wrpmeta

// Source represents some source of string key/value pairs that are used to
// copy into metadata.
type Source interface {
	// GetString returns the (possibly converted) string value of a given key.
	// This method will return true to indicate that key was found in the source.
	// A false return indicates either (1) the key was not in this source, or (2) the
	// key's value could not be converted to a string.
	GetString(key string) (value string, success bool)
}

// Field describes a single key/value to copy from a source into metadata
type Field struct {
	// From is the key in a Source object to copy.  This field is required, and if not set
	// it will result in a blank key being passed to Source.GetString.
	From string

	// To is the key in the final metadata map that a corresponding value in a Source will
	// be associated with.  This field is optional.  If unset, the From key is used in
	// the metadata.
	To string

	// Default is the default value to use if no such From key exists in the source.  If unset,
	// then missing fields in the Source will also be missing in the resulting metadata.  If set,
	// then this value will be used in the metadata if Source.GetString returns false.
	Default string
}

// Builder is a fluent strategy for creating metadata for WRP messages
type Builder interface {
	// Apply copies any number of fields from a Source into the final metadata.
	// Source may be nil or no fields may be passed, which in either case results
	// in a no-op.
	Apply(Source, ...Field) Builder

	// Set sets an arbitrary key/value pair into the final metadata
	Set(key, value string) Builder

	// Add allows concatenation of builder products.  The output of Build may be passed
	// as is to this method, in addition to calling this method directly with arbitrary metadata.
	// The given map will be copied into this Builder's current product.  If allFieldsPresent is true,
	// the internal tracking of fields for this Builder is unaffected.  If, however, allFieldsPresent is false,
	// this Builder will also count the product as missing fields when its Build method returns.
	Add(product map[string]string, allFieldsPresent bool) Builder

	// Build returns the current metadata along with a flag indicating whether all fields in Apply calls
	// were found in sources.  The returned product metadata is not a copy, and will change if this Builder
	// is used to set key/value pairs again.
	Build() (product map[string]string, allFieldsPresent bool)
}

// NewBuilder creates a new builder with an empty product.
func NewBuilder() Builder {
	return &builder{
		product:          make(map[string]string),
		allFieldsPresent: true,
	}
}

// builder is the internal Builder implementation
type builder struct {
	product          map[string]string
	allFieldsPresent bool
}

func (b *builder) Apply(source Source, fields ...Field) Builder {
	if source != nil {
		for _, f := range fields {
			var (
				value, present = source.GetString(f.From)
				to             = f.To
			)

			b.allFieldsPresent = b.allFieldsPresent || present

			if len(to) == 0 {
				to = f.From
			}

			if present {
				b.product[to] = value
			} else if len(f.Default) > 0 {
				b.product[to] = f.Default
			}
		}
	}

	return b
}

func (b *builder) Add(product map[string]string, allFieldsPresent bool) Builder {
	if !allFieldsPresent {
		b.allFieldsPresent = false
	}

	for key, value := range product {
		b.product[key] = value
	}

	return b
}

func (b *builder) Set(key, value string) Builder {
	b.product[key] = value
	return b
}

func (b builder) Build() (map[string]string, bool) {
	return b.product, b.allFieldsPresent
}
