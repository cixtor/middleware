package middleware

type privTrie struct {
	root *privTrieNode
}

type privTrieNode struct {
	children [128]*privTrieNode
	isTheEnd bool
}

func newPrivTrie() *privTrie {
	return &privTrie{root: &privTrieNode{}}
}

func (t *privTrie) Insert(endpoint string) {
	node := t.root
	for i := 0; i < len(endpoint); i++ {
		charIndex := endpoint[i]
		if node.children[charIndex] == nil {
			node.children[charIndex] = &privTrieNode{}
		}
		node = node.children[charIndex]
	}
	node.isTheEnd = true
}

func (t *privTrie) Search(endpoint string) bool {
	node := t.root
	for i := 0; i < len(endpoint); i++ {
		charIndex := endpoint[i]
		if node.children[charIndex] == nil {
			return false
		}
		node = node.children[charIndex]
	}
	return node.isTheEnd
}
