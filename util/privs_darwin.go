//go:build darwin

package util

import (
	"log"
	"os"
	"strconv"
	"syscall"
)

func DropPrivs() {
	uid, _ := strconv.Atoi(os.Getenv("PUID"))
	gid, _ := strconv.Atoi(os.Getenv("PGID"))

	log.Printf("Startup IDs: UID = %d, GID = %d", syscall.Getuid(), syscall.Getgid())

	if gid > 0 {
		err := syscall.Setregid(gid, gid)
		if err != nil {
			log.Printf("Error dropping GID: %v", err)
		}
	}

	if uid > 0 {
		err := syscall.Setreuid(uid, uid)
		if err != nil {
			log.Printf("Error dropping UID: %v", err)
		}
	}

	log.Printf("Runtime IDs: UID = %d, GID = %d", syscall.Getuid(), syscall.Getgid())
}
