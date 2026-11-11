// Command cloud-shell isolates cloud CLI configuration per customer profile.
package main

import (
	"os"

	"github.com/wcampos/cloud-shell/internal/cli"
)

// version is set with -ldflags "-X main.version=..." for a release build.
var version = "dev"

func main() {
	os.Exit(cli.Main(version))
}
