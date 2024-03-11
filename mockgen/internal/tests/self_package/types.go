package core

//go:generate mockgen -package core -self_package github.com/pableeee/implgen/mockgen/internal/tests/self_package -destination mock.go github.com/pableeee/implgen/mockgen/internal/tests/self_package Methods

type Info struct{}

type Methods interface {
	getInfo() Info
}
