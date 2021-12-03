package middleware

import (
	"net/http"
)

// sep represents the endpoint folder separator.
//
// Example:
//
//	/lorem/ipsum/dolor/sit/amet
//	^     ^     ^     ^   ^
var sep byte = '/'

// nps stands for Named Parameter Symbol.
//
// Example:
//
//	/lorem/ipsum/:dolor/sit/amet
//	             ^^^^^^ this is a NPS
var nps byte = ':'

// all represents any number of character after the folder separator.
//
// Example:
//
//	/lorem/ipsum/*/dolor/sit/amet
//	              ^^^^^^^^^^^^^^^ this are not inserted in the trie.
var all byte = '*'

type privTrie struct {
	root *privTrieNode
}

type privTrieNode struct {
	children  map[byte]*privTrieNode
	parameter string
	isTheEnd  bool
	handler   http.Handler
}

func newPrivTrie() *privTrie {
	return &privTrie{root: newPrivTrieNode()}
}

func newPrivTrieNode() *privTrieNode {
	return &privTrieNode{children: make(map[byte]*privTrieNode)}
}

func (t *privTrie) Insert(endpoint string, fn http.Handler) {
	node := t.root
	total := len(endpoint)
	for i := 0; i < total; i++ {
		char := endpoint[i]
		param := ""
		if char == nps && endpoint[i-1] == sep {
			j := i + 1
			for ; j < total && endpoint[j] != sep; j++ {
				// Consume all characters that follow a colon until we find the
				// next forward slash or the end of the endpoint. Then, select
				// those characters and use them as the parameter name.
			}
			param = endpoint[i+1 : j]
			i += len(param)
		}
		if node.children[char] == nil {
			// Initialize a trie for this specific character.
			node.children[char] = newPrivTrieNode()
		}
		if param != "" {
			// Write the parameter name, if available.
			node.children[char].parameter = param
		}
		node = node.children[char]
		if char == all && endpoint[i-1] == sep {
			// If the character is an asterisk and the previous character is a
			// URL separator, commonly a forward slash, then stop inserting new
			// nodes and mark this character the end of the URL.
			break
		}
	}
	node.isTheEnd = true
	node.handler = fn
}

func (t *privTrie) Search(endpoint string) (bool, http.Handler, map[string]string) {
	node := t.root
	total := len(endpoint)
	params := map[string]string{}

	for i := 0; i < total; i++ {
		char := endpoint[i]

		// If the character we are evaluating in the URL path exists under this
		// specific node. If yes, it may be possible to continue down the tree
		// with the assumption that there is a valid static endpoint. Move to
		// the next node to verify.
		//
		// For example, consider these two routes:
		//
		//   A. /lorem/ipsum/:page/sit/amet
		//   B. /lorem/ipsum/dolor/sit/amet
		//
		// And these two requests:
		//
		//   1. /lorem/ipsum/dolor/sit/amet
		//   2. /lorem/ipsum/maker/sit/amet
		//
		// Request [1] perfectly matches the route [A], but there is another,
		// more specific, route defined as [B] that also matches the endpoint.
		// For the sake of precision, the algorithm considers exact matches
		// first before checking for parameterized URL segments.
		//
		// Request [2], however, does not match route [B] but matches route [A]
		// and that is the one the algorithm selects to continue checking for
		// the other URL segments.
		if node.children[char] != nil {
			node = node.children[char]
			continue
		}

		// Check if there is a parameterized URL segment under this node.
		if node.children[nps] != nil {
			j := i
			for ; j < total && endpoint[j] != sep; j++ {
				// Consume all characters between the colon and the next slash.
				//
				// For example, if a route is defined as:
				//
				//   A. /lorem/ipsum/:page/sit/amet
				//
				// And the endpoint we are searching is:
				//
				//   1. /lorem/ipsum/some-page-name/sit/amet
				//
				// Then, the for loop is supposed to consume all these letters:
				//
				//   1. /lorem/ipsum/some-page-name/sit/amet
				//                   ^^^^^^^^^^^^^^
				//
				// Then, the function stores the consumed characters inside the
				// params variable as "page=some-page-name". Finally, it moves
				// the cursor N positions to the right, where N is the number
				// of characters in the parameter value.
			}
			value := endpoint[i:j]
			i += len(value) - 1
			params[node.children[nps].parameter] = value
			node = node.children[nps]
			continue
		}

		if node.children[all] != nil {
			node = node.children[all]
			break
		}

		return false, nil, nil
	}

	if total == 1 && endpoint[0] == sep && node.children[all] != nil {
		// The root node is a special case, especially when using an asterisk.
		// For example, if we define a route like the one below:
		//
		//   A. /*
		//
		// All the following URLs match as expected:
		//
		//   1. /hello
		//   2. /hello/
		//   3. /hello/world
		//   4. /hello/world/
		//   5. /hello/world/how-are-you
		//   6. /hello/world/how-are-you/
		//
		// However, when we try to access "/" the for loop below does not work
		// because the implementation is looking for a specific character to
		// match when searching for nodes, and when searching for the root node
		// at "/", there is no character to match.
		//
		// This condition handles this edge case.
		return node.children[all].isTheEnd, node.children[all].handler, params
	}

	return node.isTheEnd, node.handler, params
}
