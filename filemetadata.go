package chunk

// Metadata is the information required to reconstruct the original file from
// its chunks.
type Metadata struct {
	TopChecksum    Sum224
	ChunkChecksums []Sum224
	Width          int64
}
