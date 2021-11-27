package middleware

// nps stands for Named Parameter Symbol.
//
// Example:
//
//	/lorem/ipsum/:dolor/sit/amet
//	             ^^^^^^ this is a NPS
var nps byte = ':'

type privTrie struct {
	root *privTrieNode
}

type privTrieNode struct {
	children map[byte]*privTrieNode
	paramKey string
	isTheEnd bool
}

func newPrivTrie() *privTrie {
	return &privTrie{root: newPrivTrieNode()}
}

func newPrivTrieNode() *privTrieNode {
	return &privTrieNode{children: make(map[byte]*privTrieNode)}
}

func (t *privTrie) Insert(endpoint string) {
	node := t.root
	total := len(endpoint)
	for i := 0; i < total; i++ {
		char := endpoint[i]
		param := []byte{}
		if char == nps {
			for j := i + 1; j < total; j++ {
				if endpoint[j:j+1] == sep {
					break
				}
				param = append(param, endpoint[j])
			}
			i += len(param)
		}
		if node.children[char] == nil {
			node.children[char] = newPrivTrieNode()
			node.children[char].paramKey = string(param)
		}
		node = node.children[char]
	}
	node.isTheEnd = true
}

func (t *privTrie) Search(endpoint string) (bool, map[string]string) {
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
			for ; j < total && endpoint[j] != sep[0]; j++ {
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
			params[node.children[nps].paramKey] = value
			node = node.children[nps]
			continue
		}

		return false, nil
	}

	return node.isTheEnd, params
}
