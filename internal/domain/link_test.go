package domain

import "testing"

func TestLinkNetworksAndImages(t *testing.T) {
	nets := []Network{
		{ID: "nid1234567890", IDShort: "nid123456789", Name: "frontend", Driver: "bridge"},
		{ID: "nid9999999999", IDShort: "nid999999999", Name: "backend", Driver: "bridge"},
	}
	imgID := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	imgs := []Image{
		{ID: imgID, IDShort: ShortID(imgID), RepoTags: []string{"nginx:1.25"}},
		{ID: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", IDShort: "bbbbbbbbbbbb", RepoTags: nil},
	}
	ctrs := []Container{
		{
			Name: "web", Stack: "prod", Image: "nginx:1.25",
			ImageID: imgID,
			Endpoints: []NetworkEndpoint{{NetworkID: "nid1234567890", NetworkName: "frontend"}},
		},
	}

	linkedN := LinkNetworks(nets, ctrs)
	var frontend Network
	for _, n := range linkedN {
		if n.Name == "frontend" {
			frontend = n
		}
	}
	if len(frontend.Containers) != 1 || frontend.Containers[0] != "web" {
		t.Fatalf("network link: %+v", frontend)
	}
	if len(frontend.Stacks) != 1 || frontend.Stacks[0] != "prod" {
		t.Fatalf("stacks: %+v", frontend.Stacks)
	}

	linkedI := LinkImages(imgs, ctrs)
	var nginx Image
	for _, im := range linkedI {
		if im.ID == imgID {
			nginx = im
		}
	}
	if nginx.ContainerCount != 1 || nginx.Containers[0] != "web" {
		t.Fatalf("image link: %+v", nginx)
	}
	var dangling Image
	for _, im := range linkedI {
		if im.IDShort == "bbbbbbbbbbbb" {
			dangling = im
		}
	}
	if !dangling.Dangling {
		t.Fatal("expected dangling")
	}
}
