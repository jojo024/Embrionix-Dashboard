// Package version holds the application's build version.
package version

// Version is the running build's version. It is injected at build time via
//
//	-ldflags "-X github.com/embrionix/dashboard/internal/version.Version=v1.2.3"
//
// and defaults to "dev" for local / un-tagged builds. A "dev" build never
// reports an available update (it cannot be compared against release tags).
var Version = "dev"

// IsRelease reports whether this build carries a real release version
// (i.e. it was built from a tag, not a local "dev" build).
func IsRelease() bool {
	return Version != "dev" && Version != ""
}
