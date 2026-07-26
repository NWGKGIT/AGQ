//go:build linux

package detector

import (
	"reflect"
	"testing"
)

func TestParseNetTCPFiltersLoopbackListenPorts(t *testing.T) {
	data := []byte(`  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1D08 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 123 1 0000000000000000 100 0 0 10 0
   1: 00000000:1D09 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 124 1 0000000000000000 100 0 0 10 0
   2: 0100007F:1D0A 00000000:0000 01 00000000:00000000 00:00000000 00000000  1000        0 125 1 0000000000000000 100 0 0 10 0
   3: 00000000000000000000000001000000:2329 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 126 1 0000000000000000 100 0 0 10 0
`)

	got := parseNetTCP(data)
	want := []int{7432, 9001}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseNetTCP() = %#v, want %#v", got, want)
	}
}
