package storage

import "io"

type Storage interface {
	SaveAttachments(srcs []io.ReadSeeker) ([]string, error)
	DeleteAttachmentFolders(keys []string)
	GetFilePath(key string, size string) (string, error)
}

const imgFull = "full.jpg"
const imgThumb = "thumbnail.jpg"
const tempDir = ".temp"

const (
	SizeFull  = "full"
	SizeThumb = "thumb"
)
