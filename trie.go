package middleware

type privTrie struct {
	root *privTrieNode
}

type privTrieNode struct {
	children map[byte]*privTrieNode
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
	for i := 0; i < len(endpoint); i++ {
		charIndex := endpoint[i]
		if node.children[charIndex] == nil {
			node.children[charIndex] = newPrivTrieNode()
		}
		node = node.children[charIndex]
	}
	node.isTheEnd = true
}

func (t *privTrie) Search(endpoint string) bool {
	node := t.root
	total := len(endpoint)
	for i := 0; i < total; i++ {
		charIndex := endpoint[i]
		if charIndex == nps[0] {
			for i < total && endpoint[i:i+1] != sep {
				// Ignore the remaining part of the endpoint string.
				i++
			}
			if i == total {
				// Continue searching the remaining part of the trie.
				break
			}
			charIndex = endpoint[i]
		}
		if node.children[charIndex] == nil {
			return false
		}
		node = node.children[charIndex]
	}
	return node.isTheEnd
}
