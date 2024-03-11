package mock_names

import (
	"testing"

	"github.com/pableeee/implgen/gomock"
	"github.com/pableeee/implgen/mockgen/internal/tests/mock_name/mocks"
	"github.com/pableeee/implgen/mockgen/internal/tests/mock_name/post"
	"github.com/pableeee/implgen/mockgen/internal/tests/mock_name/user"
)

func TestMockNames(t *testing.T) {
	ctrl := gomock.NewController(t)

	userService := mocks.NewUserServiceMock(ctrl)
	postService := mocks.NewPostServiceMock(ctrl)

	gomock.InOrder(
		userService.EXPECT().
			Create("John Doe").
			Return(&user.User{Name: "John Doe"}, nil),
		postService.EXPECT().
			Create(gomock.Eq("test title"), gomock.Eq("test body"), gomock.Eq(&user.User{Name: "John Doe"})).
			Return(&post.Post{
				Title: "test title",
				Body:  "test body",
				Author: &user.User{
					Name: "John Doe",
				},
			}, nil))
	u, err := userService.Create("John Doe")
	if err != nil {
		t.Fatal("unexpected error")
	}

	p, err := postService.Create("test title", "test body", u)
	if err != nil {
		t.Fatal("unexpected error")
	}

	if p.Title != "test title" || p.Body != "test body" || p.Author.Name != u.Name {
		t.Fatal("unexpected postService.Create result")
	}
}
