package gomock

import "fmt"

type mockInstance interface {
	ISGOMOCK() struct{}
}
type mockedStringer interface {
	fmt.Stringer
	mockInstance
}

// getString is a safe way to convert a value to a string for printing results
// If the value is a a mock, getString avoids calling the mocked String() method,
// which avoids potential deadlocks
func getString(x any) string {
	switch v := x.(type) {
	case mockedStringer:
		return fmt.Sprintf("%T", v)
	case fmt.Stringer:
		return safeString(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// safeString calls String() with panic recovery to handle nil receivers
func safeString(s fmt.Stringer) (result string) {
	defer func() {
		if r := recover(); r != nil {
			// If String() panicked (e.g., nil receiver), use fmt.Sprintf instead
			result = fmt.Sprintf("%v", s)
		}
	}()
	return s.String()
}
