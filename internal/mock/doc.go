/*
Package mock contains mock implementations of various interfaces, intended
for use in unit-tests.

There are three kinds of mocks:

  1. Mocks of interfaces defined in Wetware,
  2. Mocks of Go standard-library interfaces; and,
  2. Mocks of interfaces defined in third-party package.

Type 1 mocks are located in `./pkg/...`.  Note that the directory structure
mirrors that of the root-level `pkg/` path.

Types 2 & 3 are defined by a `.go` file in `./`, which contains interface
definitions that embed the third-party interface to be mocked.  The mock-
implementation is then located in a directory under `./` of the same name.
As an example, mocks for the standard-library "io" package, are defined in
`./io.go` and the generated output is found in `./io/io.go`.

The package name of all mock implementations follows the `mock_*` pattern,
where `*` is the original package name.  In the above example, the package
name is `mock_io`.
*/
package mock
