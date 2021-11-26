package middleware

import (
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
			if root.Search(tc.query) != tc.found {
				t.Fatalf("searching for %s should return %#v", tc.query, tc.found)
			}
		})
	}
}
