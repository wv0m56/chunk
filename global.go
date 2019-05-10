package chunk

import "errors"

const (
	readBufferSize  = 1024 * 1024 // 1MB
	writeBufferSize = 1024 * 1024
)

var (
	errStreamStillRunning    = errors.New("input stream still running")
	errResubmitSameChunk     = errors.New("processed chunk submitted again")
	errNoChunkInMetadata     = errors.New("chunk not registered in metadata")
	errFinishedReconstructor = errors.New("finished reconstructor")
	errChunkChecksum         = errors.New("chunk checksum error")
)
