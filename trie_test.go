package middleware

import (
	"reflect"
	"testing"
)

func TestTrieBasic(t *testing.T) {
	root := newPrivTrie()

	root.Insert("/")
	root.Insert("/hello")
	root.Insert("/hello/world")
	root.Insert("/hello/world/how")
	root.Insert("/hello/world/how/are")
	root.Insert("/hello/world/how/are/you")

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
			if wasFound, _ := root.Search(tc.query); wasFound != tc.found {
				t.Fatalf("searching for %s should return %#v", tc.query, tc.found)
			}
		})
	}
}

func TestTrieWithNamedParameters(t *testing.T) {
	root := newPrivTrie()

	root.Insert("/")
	root.Insert("/home")
	root.Insert("/about")
	root.Insert("/contact-us")
	root.Insert("/blog")
	root.Insert("/blog/:postid")
	root.Insert("/products")
	root.Insert("/products/:itemid")
	root.Insert("/login")
	root.Insert("/register")
	root.Insert("/user/:username/profile")
	root.Insert("/user/settings")
	root.Insert("/user/settings/:pageid/success")
	root.Insert("/user/orders")
	root.Insert("/user/orders/:orderid")

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
	}
	for _, tc := range testCases {
		t.Run(tc.webpage, func(t *testing.T) {
			wasFound, params := root.Search(tc.webpage)
			if wasFound != tc.found {
				t.Fatalf("searching for %s should return %#v", tc.webpage, tc.found)
			}
			if tc.found && !reflect.DeepEqual(params, tc.params) {
				t.Errorf("Search(%q) returned %#v, expected %#v", tc.webpage, params, tc.params)
			}
		})
	}
}
