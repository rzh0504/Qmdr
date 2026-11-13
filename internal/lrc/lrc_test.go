package lrc

import "testing"

func TestMergeAppendsTranslations(t *testing.T) {
	original := "[ti:Song]\n[00:01.00]Hello\n[00:03.00]World\n[00:05.00]End"
	translation := "[ti:Song]\n[00:01.00]你好\n[00:03.00]世界\n[00:05.00]//"
	got := Merge(original, translation)
	want := "[ti:Song]\n[00:01.00]Hello\n[00:01.00]你好\n[00:03.00]World\n[00:03.00]世界\n[00:05.00]End"
	if got != want {
		t.Fatalf("Merge =\n%s\nwant\n%s", got, want)
	}
}

func TestMergeToleratesSmallOffsets(t *testing.T) {
	original := "[00:10.94]Line"
	translation := "[00:10.93]译文"
	got := Merge(original, translation)
	want := "[00:10.94]Line\n[00:10.94]译文"
	if got != want {
		t.Fatalf("Merge = %q, want %q", got, want)
	}
}

func TestMergeWithoutTranslations(t *testing.T) {
	original := "[00:01.00]Hello"
	for _, translation := range []string{"", "[ti:Song]", "[00:01.00]//", "[00:01.00]"} {
		if got := Merge(original, translation); got != original {
			t.Fatalf("Merge(%q) = %q, want the original", translation, got)
		}
	}
}

func TestMergeReusesOriginalTimestampFormat(t *testing.T) {
	got := Merge("[00:01.500]Hello", "[00:01.500]你好")
	want := "[00:01.500]Hello\n[00:01.500]你好"
	if got != want {
		t.Fatalf("Merge = %q, want %q", got, want)
	}
}
