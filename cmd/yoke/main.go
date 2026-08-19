// yoke - The yoke that binds your tasks to your work.
package main

import "github.com/ashvinbhat/yoke/internal/cli"

// version is stamped at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
