package text

import "sync"

// FileId is a compact handle for a source file registered with a [FileSet].
// Like [Symbol], it is only meaningful relative to the FileSet that produced
// it. The zero value is [NoFile], a sentinel that names no file.
type FileId uint32

// NoFile is the zero [FileId]; it refers to no file. Real files registered with
// a [FileSet] always have a non-zero FileId.
const NoFile FileId = 0

// FileSet maps source-file paths to compact [FileId] handles and back. It is
// safe for concurrent use by multiple goroutines.
type FileSet struct {
	mu    sync.RWMutex
	ids   map[string]FileId
	paths []string // paths[id]; index 0 is the empty NoFile slot
}

// NewFileSet returns an empty FileSet.
func NewFileSet() *FileSet {
	return &FileSet{
		ids:   make(map[string]FileId),
		paths: []string{""}, // reserve index 0 for NoFile
	}
}

// Add registers path and returns its [FileId], assigning a new one the first
// time path is seen. Registering the same path again returns the same FileId.
// The returned FileId is always non-zero.
func (fs *FileSet) Add(path string) FileId {
	fs.mu.RLock()
	id, ok := fs.ids[path]
	fs.mu.RUnlock()
	if ok {
		return id
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()
	if id, ok := fs.ids[path]; ok {
		return id
	}
	id = FileId(len(fs.paths))
	fs.paths = append(fs.paths, path)
	fs.ids[path] = id
	return id
}

// Lookup returns the [FileId] previously assigned to path and whether it was
// registered. It never registers a new path.
func (fs *FileSet) Lookup(path string) (FileId, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	id, ok := fs.ids[path]
	return id, ok
}

// Path returns the path registered for id. [NoFile] resolves to the empty
// string. It panics if id did not come from this FileSet (i.e. is out of
// range).
func (fs *FileSet) Path(id FileId) string {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	if int(id) >= len(fs.paths) {
		panic("text: FileId out of range for this FileSet")
	}
	return fs.paths[id]
}

// Len reports the number of files registered, not counting the [NoFile]
// sentinel.
func (fs *FileSet) Len() int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return len(fs.paths) - 1
}
