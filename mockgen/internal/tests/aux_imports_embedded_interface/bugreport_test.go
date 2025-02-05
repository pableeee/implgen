package bugreport

import (
	"testing"

	"github.com/pableeee/implgen/gomock"
)

// TestValidInterface assesses whether or not the generated mock is valid
func TestValidInterface(t *testing.T) {
	ctrl := gomock.NewController(t)

	s := NewMockSource(ctrl)
	s.EXPECT().Method().Return("")

	CallForeignMethod(s)
}
