package llama

// pieceFunc fills p with a token's text. It returns the number of bytes
// written, or -(required size) if p is too small (llama_token_to_piece contract).
type pieceFunc func(p []byte) int

// readPiece returns the token text and the buffer to reuse for the next token.
// The buffer only grows when the token doesn't fit.
func readPiece(buf []byte, fill pieceFunc) (text, newBuf []byte) {
	n := fill(buf)
	if n < 0 {
		buf = make([]byte, -n)
		n = fill(buf)
	}
	if n <= 0 {
		return nil, buf
	}
	return buf[:n], buf
}
