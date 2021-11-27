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

func (t *privTrie) Search(endpoint string) bool {
	node := t.root
	total := len(endpoint)
	for i := 0; i < total; i++ {
		char := endpoint[i]
		// If the character we are evaluating in the URL path exists under this
		// specific node. If yes, it may be possible to continue down the tree
		// with the assumption that there is a valid static endpoint. Move to
		// the next node to verify.
		if node.children[char] != nil {
			node = node.children[char]
			continue
		}
		// If the character does not exists under the node but a colon does,
		// then assume that we have a dynamic URL segment. Read the text until
		// the next forward slash and use it as the parameter value.
		if node.children[nps] != nil {
			value := []byte{}
			for j := i; j < total; j++ {
				if endpoint[j:j+1] == sep {
					break
				}
				value = append(value, endpoint[j])
			}
			i += len(value) - 1
			node = node.children[nps]
			continue
		}
		// At this point, it is safe to say the URL path is not defined.
		return false
	}
	return node.isTheEnd
}
