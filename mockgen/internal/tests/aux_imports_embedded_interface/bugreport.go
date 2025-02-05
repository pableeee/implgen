package bugreport

// Reproduce the issue described in the README.md with
// $ go:generate mockgen -aux_files faux=faux/faux.go -destination bugreport_mock.go -package bugreport -source=bugreport.go Example

import (
	"log"

	"github.com/pableeee/implgen/mockgen/internal/tests/aux_imports_embedded_interface/faux"
)

// Source is an interface w/ an embedded foreign interface
type Source interface {
	faux.Foreign
}

func CallForeignMethod(s Source) {
	log.Println(s.Method())
}
