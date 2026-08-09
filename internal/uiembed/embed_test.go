package uiembed

import "testing"

func TestAvailable(t *testing.T) {
	if !Available() {
		t.Fatal("expected embedded index.html")
	}
	fsys, err := FS()
	if err != nil {
		t.Fatal(err)
	}
	f, err := fsys.Open("index.html")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
}
