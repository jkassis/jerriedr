package core

// VOID is a type that uses no space. Useful for...
// 1. sentinal / signal channels
// 2. maps used as sets
type VOID struct{}

var Void VOID
