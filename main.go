package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	wmClose = 0x10
)

var (
	user32             = syscall.MustLoadDLL("user32.dll")
	procEnumWindows    = user32.MustFindProc("EnumWindows")
	procGetWindowTextW = user32.MustFindProc("GetWindowTextW")
	procPostMessageW   = user32.MustFindProc("PostMessageW")
)

func EnumWindows(enumFunc uintptr, lparam uintptr) error {
	_, _, err := syscall.Syscall(procEnumWindows.Addr(), 2, enumFunc, lparam, 0)
	if err != 0 {
		return fmt.Errorf("EnumWindows failed: %w", err)
	}
	return nil
}

func GetWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (int32, error) {
	n, _, err := syscall.Syscall(procGetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	if n == 0 && err != 0 {
		return 0, fmt.Errorf("GetWindowText failed: %w", err)
	}
	return int32(n), nil
}

func FindWindow(title string) (syscall.Handle, error) {
	var hwnd syscall.Handle
	cb := syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
		b := make([]uint16, 200)
		_, err := GetWindowText(h, &b[0], int32(len(b)))
		if err != nil {
			return 1 // continue enumeration
		}
		if syscall.UTF16ToString(b) == title {
			hwnd = h
			return 0 // stop enumeration
		}
		return 1 // continue enumeration
	})
	if err := EnumWindows(cb, 0); err != nil {
		return 0, err
	}
	if hwnd == 0 {
		return 0, fmt.Errorf("no window with title '%s' found", title)
	}
	return hwnd, nil
}

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatalf("Usage: %s <window title>", args[0])
	}
	title := args[1]

	var h *syscall.Handle
	for i := 0; i < 60; i++ {
		hd, err := FindWindow(title)
		if err != nil {
			log.Printf("Failed to find window: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		h = &hd
		log.Printf("Found '%s' window: handle=0x%x\n", title, h)
		break
	}
	// Close window
	_, _, err := syscall.Syscall(procPostMessageW.Addr(), 3, uintptr(*h), uintptr(wmClose), 0)
	log.Printf("Close window '%s': %v", title, err)
}
