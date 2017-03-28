package api

import (
	"math/rand"
	"time"
)

// File represents a file upload.
type File struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// charset represents the ID generator character set.
const charset = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnopqrstuvwxyz23456789"

// setName sets a pseudo-random name using the correct extension.
func (f *File) setName(ext string) {
	b := make([]byte, 5)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	f.Name = string(b) + ext
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
