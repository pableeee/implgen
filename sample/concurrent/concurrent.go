// Package concurrent demonstrates how to use gomock with goroutines.
package concurrent

//go:generate mockgen -destination mock/concurrent_mock.go github.com/pableeee/implgen/sample/concurrent Math

type Math interface {
	Sum(a, b int) int
}
