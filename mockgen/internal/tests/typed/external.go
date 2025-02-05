package typed

import (
	"github.com/pableeee/implgen/mockgen/internal/tests/typed/other"
	"golang.org/x/exp/constraints"
)

//go:generate mockgen --source=external.go --destination=source/mock_external_test.go --package source -typed

type ExternalConstraint[I constraints.Integer, F constraints.Float] interface {
	One(string) string
	Two(I) string
	Three(I) F
	Four(I) Foo[I, F]
	Five(I) Baz[F]
	Six(I) *Baz[F]
	Seven(I) other.One[I]
	Eight(F) other.Two[I, F]
	Nine(Iface[I])
	Ten(*I)
}
