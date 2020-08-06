package mocktest

/*
Package mocktest contains mocks and associated utilities for unit testing.

The directory structure of mock mirrors that of the ww package, with e.g. mocks of
public interfaces located under `pkg/`.  In addition to this mirrored hierarchy, the
mocktest package contains the following directories:

(1) vendor/

	The `vendor/` directory contains auto-generated mocks for 3rd party interfaces
	used internally by `ww`.  These mocks are generated from the `vendor.go` file.

*/
