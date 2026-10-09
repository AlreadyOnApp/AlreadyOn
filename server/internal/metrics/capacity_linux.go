//go:build linux

package metrics

import "syscall"

// capacity matches df: used is total minus free blocks, and the total excludes
// root-reserved blocks unprivileged users cannot allocate. statfs reports the
// mounted filesystem itself, so bind and NFS mounts work inside a container.
// dev identifies the filesystem so folders sharing one are counted once.
func capacity(path string) (used, total int64, dev uint64, ok bool) {
	var st syscall.Statfs_t
	var info syscall.Stat_t
	if path == "" || syscall.Statfs(path, &st) != nil || st.Blocks == 0 || syscall.Stat(path, &info) != nil {
		return 0, 0, 0, false
	}
	size := int64(st.Bsize)
	used = int64(st.Blocks-st.Bfree) * size
	return used, used + int64(st.Bavail)*size, uint64(info.Dev), true
}
