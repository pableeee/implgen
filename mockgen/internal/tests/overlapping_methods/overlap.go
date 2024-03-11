package overlap

//go:generate mockgen -package overlap -destination mock.go -source overlap.go -aux_files github.com/pableeee/implgen/mockgen/internal/tests/overlapping_methods=interfaces.go

type ReadWriteCloser interface {
	ReadCloser
	WriteCloser
}
