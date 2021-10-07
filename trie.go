package middleware

type trie struct {
	children  map[rune]*trie
	endOfWord bool
}

func newTrie() *trie {
	return &trie{
		children: make(map[rune]*trie),
	}
}

func (t *trie) Insert(word string) {
	curr := t

	for _, c := range word {
		if curr.children[c] == nil {
			curr.children[c] = newTrie()
		}

		curr = curr.children[c]
	}

	curr.endOfWord = true
}

func (t *trie) Search(word string) bool {
	curr := t

	for _, c := range word {
		if curr.children[c] == nil {
			return false
		}

		curr = curr.children[c]
	}

	return curr.endOfWord
}
