package middleware

import (
	"strings"
)

type trie struct {
	root *trieNode
}

func newTrie() *trie {
	return &trie{
		root: newTrieNode(),
	}
}

type trieNode struct {
	children  map[string]*trieNode
	endOfWord bool
}

func newTrieNode() *trieNode {
	return &trieNode{
		children: make(map[string]*trieNode),
	}
}

func (t *trie) Insert(urlPath string) {
	node := t.root
	folders := strings.Split(urlPath, sep)

	for _, folder := range folders {
		if folder == "" {
			continue
		}

		if _, ok := node.children[folder]; !ok {
			node.children[folder] = newTrieNode()
		}

		node = node.children[folder]
	}

	node.endOfWord = true
}

func (t *trie) Search(urlPath string) bool {
	node := t.root
	folders := strings.Split(urlPath, sep)

	for _, folder := range folders {
		if folder == "" {
			continue
		}

		if _, ok := node.children[folder]; !ok {
			return false
		}

		node = node.children[folder]
	}

	return node.endOfWord
}
