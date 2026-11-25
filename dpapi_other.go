//go:build !windows

package main

// Trên hệ điều hành khác (chỉ dùng khi phát triển, thử nghiệm): không mã hóa.
func protectData(b []byte) ([]byte, error)   { return b, nil }
func unprotectData(b []byte) ([]byte, error) { return b, nil }
