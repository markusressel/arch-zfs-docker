package builder

// NewFakeSystem lets external tests (package builder_test, which may import packages that depend on
// builder) run builds against the fake system.
var NewFakeSystem = newFakeSystem
