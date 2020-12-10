// Package mem implements the wetware memory model, providing data and type primitives.
// It defines an high-level API on top of the `api` package, used for for sharing data
// across package boundaries.
//
// The mem package is organized around two basic types:  Type and Value.  Type is an
// alias of api.Any_Which, and identifies the type of a Value.  Value is a wrapper
// around api.Any, providing a mid-level API for manipulating wetware values that sits
// between the `api` package (level 0) and the specific datatypes implemented in the
// `lang` package (level 2).
package mem
