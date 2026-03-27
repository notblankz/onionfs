package ui

import "fmt"

const Version = "0.1.0"

func PrintHelp() {
	fmt.Printf(`
OnionFS - a union filesystem built on FUSE

Usage:
  onionfs -l <lower> -u <upper> -m <mountpoint> [flags]

Required:
  -l, --lower       <path>   lower (read-only) directory
  -u, --upper       <path>   upper (read-write) directory
  -m, --mountpoint  <path>   mount point

Behaviour:
      --no-cow               disable copy-on-write
      --show-meta            show .wh.* files in directory listings

Mode:
      --daemon               run in background (default: foreground)

Other:
  -h, --help                 print this help
      --version              print version and exit
`)
}

func PrintVersion() {
	fmt.Printf("OnionFS Version %s", Version)
}
