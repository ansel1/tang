#!/bin/bash
mkdir -p timepkg1 timepkg2
cat << 'GO' > timepkg1/dummy_test.go
package timepkg1
import ("testing"; "time")
func Test1(t *testing.T) { time.Sleep(1 * time.Second) }
GO
cat << 'GO' > timepkg2/dummy_test.go
package timepkg2
import ("testing"; "time")
func Test2(t *testing.T) { time.Sleep(1 * time.Second) }
GO
go test -json -p 1 ./timepkg1 ./timepkg2
rm -rf timepkg1 timepkg2
