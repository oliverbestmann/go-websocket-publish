package ws

import "sync"

type StreamId string

type TokenManager struct {
	lock   sync.RWMutex
	tokens map[Token]StreamId
}

func NewTokenManager() *TokenManager {
	return &TokenManager{
		lock: sync.RWMutex{},
		tokens: make(map[Token]StreamId),
	}
}

func (tm *TokenManager) Allowed(stream StreamId, token Token) bool {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	return tm.tokens[token] == stream
}

func (tm *TokenManager) Allow(stream StreamId, token Token) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	// allow this token.
	tm.tokens[token] = stream
}

func (tm *TokenManager) Revoke(stream StreamId, token Token) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.tokens[token] == stream {
		// remove the token!
		delete(tm.tokens, token)
	}
}

func (tm *TokenManager) RemoveStream(streamToRemove StreamId) []Token {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tokensToClose := make([]Token, 0)
	for token, stream := range tm.tokens {
		if stream == streamToRemove {
			tokensToClose = append(tokensToClose, token)
			delete(tm.tokens, token)
		}
	}

	return tokensToClose
}
