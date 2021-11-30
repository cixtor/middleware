package middleware

import (
	"reflect"
	"testing"
)

func TestTrieBasic(t *testing.T) {
	root := newPrivTrie()

	root.Insert("/", nil)
	root.Insert("/hello", nil)
	root.Insert("/hello/world", nil)
	root.Insert("/hello/world/how", nil)
	root.Insert("/hello/world/how/are", nil)
	root.Insert("/hello/world/how/are/you", nil)

	testCases := []struct {
		found bool
		query string
	}{
		{found: true, query: "/hello/world/how/are/you"}, /* <<< */
		{found: false, query: "/hello/world/how/are/yo"},
		{found: false, query: "/hello/world/how/are/y"},
		{found: false, query: "/hello/world/how/are/"},
		{found: true, query: "/hello/world/how/are"}, /* <<< */
		{found: false, query: "/hello/world/how/ar"},
		{found: false, query: "/hello/world/how/a"},
		{found: false, query: "/hello/world/how/"},
		{found: true, query: "/hello/world/how"}, /* <<< */
		{found: false, query: "/hello/world/ho"},
		{found: false, query: "/hello/world/h"},
		{found: false, query: "/hello/world/"},
		{found: true, query: "/hello/world"}, /* <<< */
		{found: false, query: "/hello/worl"},
		{found: false, query: "/hello/wor"},
		{found: false, query: "/hello/wo"},
		{found: false, query: "/hello/w"},
		{found: false, query: "/hello/"},
		{found: true, query: "/hello"}, /* <<< */
		{found: false, query: "/hell"},
		{found: false, query: "/hel"},
		{found: false, query: "/he"},
		{found: false, query: "/h"},
		{found: true, query: "/"}, /* <<< */
	}
	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			if wasFound, _, _ := root.Search(tc.query); wasFound != tc.found {
				t.Fatalf("searching for %s should return %#v", tc.query, tc.found)
			}
		})
	}
}

func TestTrieWithNamedParameters(t *testing.T) {
	root := newPrivTrie()

	root.Insert("/", nil)
	root.Insert("/home", nil)
	root.Insert("/about", nil)
	root.Insert("/contact-us", nil)
	root.Insert("/blog", nil)
	root.Insert("/blog/:postid", nil)
	root.Insert("/products", nil)
	root.Insert("/products/:itemid", nil)
	root.Insert("/login", nil)
	root.Insert("/register", nil)
	root.Insert("/user/:username/profile", nil)
	root.Insert("/user/settings", nil)
	root.Insert("/user/settings/:pageid/success", nil)
	root.Insert("/user/orders", nil)
	root.Insert("/user/orders/:orderid", nil)
	root.Insert("/hello/world/how/are/you/doing", nil)
	root.Insert("/hello/world/my/name/is/:name", nil)
	root.Insert("/hello/world", nil)
	root.Insert("/hello/world/:company", nil)
	root.Insert("/foo/bar", nil)
	root.Insert("/something/interesting/to/render", nil)
	root.Insert("/something/interesting/for/everyone", nil)
	root.Insert("/users/:id", nil)
	root.Insert("/articles/:slug/comments/:id", nil)
	root.Insert("/books/:isbn/chapters/:chapterNumber/pages/:pageNumber", nil)

	testCases := []struct {
		found   bool
		webpage string
		params  map[string]string
	}{
		{found: true, webpage: "/", params: map[string]string{}},
		{found: false, webpage: "/notfound"},
		{found: true, webpage: "/home", params: map[string]string{}},
		{found: true, webpage: "/about", params: map[string]string{}},
		{found: true, webpage: "/contact-us", params: map[string]string{}},
		{found: true, webpage: "/blog", params: map[string]string{}},
		{found: false, webpage: "/blog/"},
		{found: true, webpage: "/blog/post-1", params: map[string]string{"postid": "post-1"}},
		{found: true, webpage: "/blog/post-2", params: map[string]string{"postid": "post-2"}},
		{found: true, webpage: "/blog/post-3", params: map[string]string{"postid": "post-3"}},
		{found: false, webpage: "/blog/post-4/hello-world"},
		{found: true, webpage: "/products", params: map[string]string{}},
		{found: false, webpage: "/products/"},
		{found: true, webpage: "/products/item-1", params: map[string]string{"itemid": "item-1"}},
		{found: true, webpage: "/products/item-2", params: map[string]string{"itemid": "item-2"}},
		{found: true, webpage: "/products/item-3", params: map[string]string{"itemid": "item-3"}},
		{found: false, webpage: "/products/item-4/foobar"},
		{found: true, webpage: "/login", params: map[string]string{}},
		{found: true, webpage: "/register", params: map[string]string{}},
		{found: true, webpage: "/user/root/profile", params: map[string]string{"username": "root"}},
		{found: false, webpage: "/user/root/profile/foobar"},
		{found: true, webpage: "/user/-/profile", params: map[string]string{"username": "-"}},
		{found: false, webpage: "/user/profile"},
		{found: true, webpage: "/user/settings", params: map[string]string{}},
		{found: false, webpage: "/user/settings/"},
		{found: false, webpage: "/user/settings/foobar"},
		{found: false, webpage: "/user/settings/foobar/"},
		{found: true, webpage: "/user/settings/foobar/success", params: map[string]string{"pageid": "foobar"}},
		{found: true, webpage: "/user/orders", params: map[string]string{}},
		{found: false, webpage: "/user/orders/"},
		{found: true, webpage: "/user/orders/order-1", params: map[string]string{"orderid": "order-1"}},
		{found: false, webpage: "/user/orders/order-1/foobar"},
		// users
		{found: false, webpage: "/users/123/hello"},
		{found: false, webpage: "/users/123/"},
		{found: true, webpage: "/users/123", params: map[string]string{"id": "123"}},
		{found: false, webpage: "/users/"},
		{found: false, webpage: "/users"},
		// articles
		{found: false, webpage: "/articles/my-article/comments/456/hello"},
		{found: false, webpage: "/articles/my-article/comments/456/"},
		{found: true, webpage: "/articles/my-article/comments/456", params: map[string]string{"slug": "my-article", "id": "456"}},
		{found: false, webpage: "/articles/my-article/comments/"},
		{found: false, webpage: "/articles/my-article/comments"},
		{found: false, webpage: "/articles/my-article/"},
		{found: false, webpage: "/articles/my-article"},
		{found: false, webpage: "/articles/"},
		{found: false, webpage: "/articles"},
		// books
		{found: false, webpage: "/books/978-0547928227/chapters/3/pages/42/hello"},
		{found: false, webpage: "/books/978-0547928227/chapters/3/pages/42/"},
		{found: true, webpage: "/books/978-0547928227/chapters/3/pages/42", params: map[string]string{"isbn": "978-0547928227", "chapterNumber": "3", "pageNumber": "42"}},
		{found: false, webpage: "/books/978-0547928227/chapters/3/pages/"},
		{found: false, webpage: "/books/978-0547928227/chapters/3/pages"},
		{found: false, webpage: "/books/978-0547928227/chapters/3/"},
		{found: false, webpage: "/books/978-0547928227/chapters/3"},
		{found: false, webpage: "/books/978-0547928227/chapters/"},
		{found: false, webpage: "/books/978-0547928227/chapters"},
		{found: false, webpage: "/books/978-0547928227/"},
		{found: false, webpage: "/books/978-0547928227"},
		{found: false, webpage: "/books/"},
		{found: false, webpage: "/books"},
	}

	for _, tc := range testCases {
		t.Run(tc.webpage, func(t *testing.T) {
			wasFound, _, params := root.Search(tc.webpage)
			if wasFound != tc.found {
				t.Fatalf("searching for %q should return %#v", tc.webpage, tc.found)
			}
			if tc.found && !reflect.DeepEqual(params, tc.params) {
				t.Fatalf("searching for %q\n- %#v\n+ %#v", tc.webpage, params, tc.params)
			}
		})
	}
}

func TestTrieAmbiguousParameter(t *testing.T) {
	root := newPrivTrie()

	root.Insert("/", nil)
	root.Insert("/user/user:name/profile", nil)

	testCases := []struct {
		found   bool
		webpage string
		params  map[string]string
	}{
		{found: true, webpage: "/user/user:name/profile", params: map[string]string{}},
		{found: false, webpage: "/user/johnsmith/profile"},
	}

	for _, tc := range testCases {
		t.Run(tc.webpage, func(t *testing.T) {
			wasFound, _, params := root.Search(tc.webpage)
			if wasFound != tc.found {
				t.Fatalf("searching for %q should return %#v", tc.webpage, tc.found)
			}
			if tc.found && !reflect.DeepEqual(params, tc.params) {
				t.Fatalf("searching for %q\n- %#v\n+ %#v", tc.webpage, params, tc.params)
			}
		})
	}
}

func TestTrieWithAsterisk(t *testing.T) {
	root := newPrivTrie()

	root.Insert("/", nil)
	root.Insert("/home", nil)
	root.Insert("/about", nil)
	root.Insert("/blog/:article", nil)
	root.Insert("/images/*", nil)
	root.Insert("/noindex/documents/*", nil)
	root.Insert("/cookies/are*delicious", nil)

	testCases := []struct {
		found   bool
		webpage string
		params  map[string]string
	}{
		{found: true, webpage: "/", params: map[string]string{}},
		{found: false, webpage: "/notfound"},
		{found: true, webpage: "/home", params: map[string]string{}},
		{found: true, webpage: "/about", params: map[string]string{}},
		{found: false, webpage: "/contact-us"},
		{found: true, webpage: "/blog/post-1", params: map[string]string{"article": "post-1"}},
		{found: true, webpage: "/blog/post-2", params: map[string]string{"article": "post-2"}},
		{found: true, webpage: "/blog/post-3", params: map[string]string{"article": "post-3"}},
		{found: false, webpage: "/images"},
		{found: false, webpage: "/images/"},
		{found: true, webpage: "/images/image-1.jpg", params: map[string]string{}},
		{found: true, webpage: "/images/image-2.png", params: map[string]string{}},
		{found: true, webpage: "/images/image-3.gif", params: map[string]string{}},
		{found: true, webpage: "/images/jpg/image-1.jpg", params: map[string]string{}},
		{found: true, webpage: "/images/png/image-2.png", params: map[string]string{}},
		{found: true, webpage: "/images/gif/image-3.gif", params: map[string]string{}},
		{found: true, webpage: "/images/sub1/image-1.jpg", params: map[string]string{}},
		{found: true, webpage: "/images/sub1/sub2/image-2.png", params: map[string]string{}},
		{found: true, webpage: "/images/sub1/sub2/sub3/image-3.gif", params: map[string]string{}},
		{found: true, webpage: "/noindex/documents/hello/world/file.pdf", params: map[string]string{}},
		{found: true, webpage: "/cookies/are*delicious", params: map[string]string{}},
		{found: false, webpage: "/cookies/are"},
	}

	for _, tc := range testCases {
		t.Run(tc.webpage, func(t *testing.T) {
			wasFound, _, params := root.Search(tc.webpage)
			if wasFound != tc.found {
				t.Fatalf("searching for %q should return %#v", tc.webpage, tc.found)
			}
			if tc.found && !reflect.DeepEqual(params, tc.params) {
				t.Fatalf("searching for %q\n- %#v\n+ %#v", tc.webpage, params, tc.params)
			}
		})
	}
}
