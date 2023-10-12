package main

import "testing"

func customSearchTest(t *testing.T, root *songFolder, searchText string, expectedSingle *songFolder) {
	actual := root.search(searchText)
	if expectedSingle == nil {
		if len(actual) != 0 {
			t.Errorf("Expected no results, got %d", len(actual))
		}
		return
	}

	if len(actual) != 1 {
		t.Fatalf("Expected 1 result, got %d for %s", len(actual), searchText)
	}

	actualSingle := actual[0]

	if actualSingle != expectedSingle {
		t.Errorf("Expected %s, got %s", expectedSingle.name, actualSingle.name)
	}
}

func TestSongFolderSearch(t *testing.T) {
	root := &songFolder{
		name:       "root",
		subFolders: []*songFolder{},
	}
	sub1 := root.addSubFolder("sub1")
	sub2 := root.addSubFolder("sub2")

	bob1 := sub1.addSubFolder("bob1")
	bob2 := sub1.addSubFolder("bob2")

	customSearchTest(t, root, "bob1", bob1)
	customSearchTest(t, root, "bob2", bob2)
	customSearchTest(t, root, "sub2", sub2)
	customSearchTest(t, root, "Bob2", bob2)
	customSearchTest(t, root, "ob1", bob1)

	// search should not return root element
	customSearchTest(t, root, "roo", nil)
}
